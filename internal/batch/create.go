package batch

import (
	"context"
	"errors"
	"net/http"
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
		if !isSlotBusy(err) {
			return t, err
		}

		r.send(WaitingSlot)
		if err := r.c.Sleep(ctx, getSlotDelay(err)); err != nil {
			return nil, err
		}
	}
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

func isSlotBusy(err error) bool {
	var apiErr *api.Error
	if !errors.As(err, &apiErr) {
		return false
	}

	if apiErr.Status != http.StatusTooManyRequests {
		return false
	}

	return api.IsCode(err, api.CodeTooManyActiveJobs, api.CodeRateLimited)
}

func getSlotDelay(err error) time.Duration {
	var apiErr *api.Error
	if !errors.As(err, &apiErr) {
		return defaultSlotWait
	}

	if apiErr.RetryAfter > 0 {
		return apiErr.RetryAfter
	}

	return defaultSlotWait
}
