package batch

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/transcript1-com/cli/internal/api"
	"github.com/transcript1-com/cli/internal/files"
	"github.com/transcript1-com/cli/internal/inputs"
)

type pool struct {
	c            *api.Client
	ins          []inputs.Input
	opt          Options
	emit         func(Event)
	full         map[int]bool
	results      []Result
	outOfMinutes atomic.Bool
	stop         context.CancelFunc
	mu           sync.Mutex
	authErr      error
}

func Run(ctx context.Context, c *api.Client, ins []inputs.Input, opt Options, emit func(Event)) ([]Result, error) {
	full, err := files.FullNames(ins)
	if err != nil {
		return nil, err
	}

	ctx, stop := context.WithCancel(ctx)
	defer stop()
	if emit == nil {
		emit = func(Event) {}
	}

	p := &pool{
		c:       c,
		ins:     ins,
		opt:     opt,
		emit:    emit,
		full:    full,
		results: make([]Result, len(ins)),
		stop:    stop,
	}
	for i, in := range ins {
		p.results[i] = Result{Index: i, Input: in}
	}

	indexes := make(chan int)
	var wg sync.WaitGroup
	for range min(max(opt.Jobs, 1), len(ins)) {
		wg.Go(func() { p.work(ctx, indexes) })
	}

	p.feed(ctx, indexes)
	wg.Wait()

	return p.results, p.authErr
}

func (p *pool) feed(ctx context.Context, indexes chan<- int) {
	defer close(indexes)
	for i := range p.ins {
		select {
		case <-ctx.Done():
			return
		case indexes <- i:
		}
	}
}

func (p *pool) work(ctx context.Context, indexes <-chan int) {
	for i := range indexes {
		p.results[i] = p.runInput(ctx, i)
	}
}

func (p *pool) runInput(ctx context.Context, i int) Result {
	res := Result{Index: i, Input: p.ins[i]}
	if ctx.Err() != nil {
		return res
	}

	if p.outOfMinutes.Load() {
		res.Status = SkippedMinutes
		p.emit(Event{
			Index:  i,
			Input:  res.Input,
			Stage:  Finished,
			Result: &res,
		})

		return res
	}

	res = runOne(ctx, p.c, i, res.Input, p.opt, p.full[i], p.send)
	p.recordResult(res)

	return res
}

func isNotStarted(ev Event) bool {
	if ev.Stage != Finished {
		return false
	}

	if ev.Result == nil {
		return false
	}

	return ev.Result.Status == NotStarted
}

func (p *pool) send(ev Event) {
	if isNotStarted(ev) {
		return
	}

	p.emit(ev)
}

func (p *pool) recordResult(res Result) {
	if res.Status != Failed {
		return
	}

	if api.IsCode(res.Err, api.CodeInsufficientMinutes, api.CodeNoActivePlan) {
		p.outOfMinutes.Store(true)
	}

	if !api.IsCode(res.Err, api.CodeUnauthorized) {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.authErr == nil {
		p.authErr = res.Err
	}

	p.stop()
}
