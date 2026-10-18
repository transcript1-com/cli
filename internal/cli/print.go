package cli

import (
	"fmt"
	"io"
	"sync"

	"github.com/transcript1-com/cli/internal/batch"
	"github.com/transcript1-com/cli/internal/inputs"
	"github.com/transcript1-com/cli/internal/output"
)

type printer struct {
	mu   sync.Mutex
	out  io.Writer
	errw io.Writer
	json bool
	err  error
}

func (p *printer) shouldPrintLine(e batch.Event) bool {
	if !p.json {
		return false
	}

	if e.Stage != batch.Finished {
		return false
	}

	return e.Result != nil
}

func (p *printer) handleEvent(e batch.Event) {
	if !p.shouldPrintLine(e) {
		return
	}

	line, err := output.JSONLine(*e.Result)
	if err != nil {
		p.fail(e.Input, err)
		return
	}

	p.write(e.Input, line)
}

func (p *printer) write(in inputs.Input, data []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.err != nil {
		return
	}

	if _, err := p.out.Write(data); err != nil {
		p.recordError(in, err)
	}
}

func (p *printer) fail(in inputs.Input, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.recordError(in, err)
}

func (p *printer) recordError(in inputs.Input, err error) {
	if p.err != nil {
		return
	}

	p.err = err
	fmt.Fprintln(p.errw, output.ErrorLine(output.Display(in), err))
}

func (p *printer) hasFailed() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.err != nil
}

func (p *printer) shouldPrintID(res batch.Result) bool {
	if res.Status != batch.Started {
		return false
	}

	if p.json {
		return false
	}

	return res.T != nil
}

func (p *printer) printResults(results []batch.Result) {
	for _, res := range results {
		if p.shouldPrintID(res) {
			p.write(res.Input, []byte(res.T.ID+" "+res.T.Status+"\n"))
		}

		if len(res.Body) > 0 {
			p.write(res.Input, res.Body)
		}
	}
}
