package batch

import (
	"context"

	"github.com/transcript1/cli/internal/api"
	"github.com/transcript1/cli/internal/inputs"
)

func Fetch(ctx context.Context, c *api.Client, id string, opt Options, emit func(Event)) Result {
	wait := waitSeconds
	if opt.NoWait {
		wait = 0
	}

	t, err := c.Get(ctx, id, wait)
	if err != nil {
		in := inputs.Input{Raw: id}
		return newRunner(c, 0, in, opt, false, emit).emitFinished(markFailed(Result{Input: in}, err))
	}

	r := newRunner(c, 0, buildInput(t), opt, false, emit)
	r.id = t.ID

	return r.emitFinished(r.handleFetched(ctx, t))
}

func (r *runner) handleFetched(ctx context.Context, t *api.Transcription) Result {
	res := Result{
		Index: r.idx,
		Input: r.in,
		T:     t,
	}
	if r.opt.NoWait {
		res.Status = Started
		return res
	}

	if path, ok := r.findExisting(); ok {
		res.Status = Skipped
		res.File = path

		return res
	}

	return r.complete(ctx, res)
}

func buildInput(t *api.Transcription) inputs.Input {
	if t.Source.Platform == "youtube" {
		in := inputs.Input{
			Raw:     t.Source.URL,
			Kind:    inputs.Link,
			URL:     t.Source.URL,
			VideoID: t.Source.MediaID,
		}

		return in
	}

	in := inputs.Input{
		Raw:  t.Source.Filename,
		Kind: inputs.File,
		Name: t.Source.Filename,
		Size: t.Source.Size,
	}

	return in
}
