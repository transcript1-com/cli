package batch

import (
	"context"
	"time"

	"github.com/transcript1/cli/internal/api"
	"github.com/transcript1/cli/internal/inputs"
	"github.com/transcript1/cli/internal/upload"
)

const defaultSlotWait = 10 * time.Second

// one key for every retry so a slot wait never makes two jobs
func (r *runner) create(ctx context.Context) (*api.Transcription, error) {
	key, err := newIdempotencyKey()
	if err != nil {
		return nil, err
	}

	for {
		t, err := r.createOnce(ctx, key)
		busy, ok := api.SlotBusy(err)
		if !ok {
			return t, err
		}

		if err := r.waitSlot(ctx, busy); err != nil {
			return nil, err
		}
	}
}

func (r *runner) waitSlot(ctx context.Context, err *api.Error) error {
	r.send(WaitingSlot)
	delay := err.RetryAfter
	if delay <= 0 {
		delay = defaultSlotWait
	}

	return r.c.Sleep(ctx, delay)
}

func (r *runner) createOnce(ctx context.Context, key string) (*api.Transcription, error) {
	if r.in.Kind == inputs.File {
		return r.upload(ctx, key)
	}

	opt := api.CreateOptions{
		Source:   r.opt.Source,
		Language: r.opt.Language,
		Diarize:  r.opt.Speakers,
	}

	return r.c.CreateLink(ctx, key, r.in.URL, opt, waitSeconds, !r.opt.Fresh)
}

func (r *runner) upload(ctx context.Context, key string) (*api.Transcription, error) {
	return upload.Run(ctx, r.c, r.in, upload.Options{
		Create:         api.CreateOptions{Language: r.opt.Language, Diarize: r.opt.Speakers},
		IdempotencyKey: key,
		Created:        func(id string) { r.id = id },
		WaitSlot:       r.waitSlot,
		Progress: func(sent, total int64) {
			r.emit(Event{
				Index: r.idx,
				Input: r.in,
				Stage: Uploading,
				Sent:  sent,
				Total: total,
			})
		},
	})
}
