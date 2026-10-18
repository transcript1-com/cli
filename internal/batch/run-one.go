package batch

import (
	"context"
	"os"

	"github.com/transcript1-com/cli/internal/api"
	"github.com/transcript1-com/cli/internal/files"
	"github.com/transcript1-com/cli/internal/inputs"
)

const waitSeconds = 60

type runner struct {
	c        *api.Client
	idx      int
	in       inputs.Input
	opt      Options
	fullName bool
	emit     func(Event)
	id       string
}

func newRunner(c *api.Client, idx int, in inputs.Input, opt Options, fullName bool, emit func(Event)) *runner {
	if emit == nil {
		emit = func(Event) {}
	}

	return &runner{
		c:        c,
		idx:      idx,
		in:       in,
		opt:      opt,
		fullName: fullName,
		emit:     emit,
	}
}

func runOne(
	ctx context.Context, c *api.Client, idx int, in inputs.Input, opt Options, fullName bool, emit func(Event),
) Result {
	r := newRunner(c, idx, in, opt, fullName, emit)
	return r.emitFinished(r.cancelUpload(r.run(ctx)))
}

func (r *runner) run(ctx context.Context) Result {
	res := Result{Index: r.idx, Input: r.in}
	if ctx.Err() != nil {
		return res
	}

	if path, ok := r.findExisting(); ok {
		res.Status = Skipped
		res.File = path

		return res
	}

	t, err := r.create(ctx)
	if err != nil {
		return r.fail(ctx, res, err)
	}

	r.id = t.ID
	res.T = t
	r.send(Queued)
	if r.opt.NoWait {
		res.Status = Started
		return res
	}

	return r.complete(ctx, res)
}

func (r *runner) complete(ctx context.Context, res Result) Result {
	t, err := r.wait(ctx, res.T)
	res.T = t
	if err != nil {
		return r.fail(ctx, res, err)
	}

	if t.Status != api.StatusCompleted {
		return markFailed(res, buildJobError(t))
	}

	r.nameLink(t)
	res.Input = r.in

	return r.writeOutput(ctx, res)
}

// a link with no local id is named after the media id, or the transcription id when the post has none
func (r *runner) nameLink(t *api.Transcription) {
	if r.in.Kind != inputs.Link {
		return
	}

	if r.in.VideoID != "" {
		return
	}

	r.in.VideoID = t.Source.MediaID
	if r.in.VideoID == "" {
		r.in.VideoID = t.ID
	}
}

func (r *runner) emitFinished(res Result) Result {
	r.emit(Event{
		Index:  r.idx,
		Input:  r.in,
		Stage:  Finished,
		Result: &res,
	})

	return res
}

func (r *runner) send(stage Stage) {
	r.emit(Event{
		Index: r.idx,
		Input: r.in,
		Stage: stage,
	})
}

func (r *runner) getFormat() string {
	if r.opt.Format == "" {
		return files.FormatText
	}

	return r.opt.Format
}

func (r *runner) findExisting() (string, bool) {
	if r.opt.Overwrite {
		return "", false
	}

	if r.opt.Dir != "" {
		return files.Existing(r.opt.Dir, r.in, r.getFormat(), r.fullName)
	}

	if r.opt.File == "" {
		return "", false
	}

	_, err := os.Stat(r.opt.File)

	return r.opt.File, err == nil
}

func markFailed(res Result, err error) Result {
	res.Status = Failed
	res.Err = err

	return res
}

func (r *runner) fail(ctx context.Context, res Result, err error) Result {
	if ctx.Err() == nil {
		return markFailed(res, err)
	}

	if res.T != nil {
		res.Status = StillRunning
		return res
	}

	res.Status = Interrupted
	if r.id != "" {
		res.T = &api.Transcription{ID: r.id, Status: api.StatusAwaitingUpload}
	}

	return res
}
