package batch

import (
	"context"
	"slices"
	"time"

	"github.com/transcript1/cli/internal/api"
)

const minPollGap = 2 * time.Second

func shouldAnnounce(announced bool, t *api.Transcription) bool {
	if announced {
		return false
	}

	return isWorking(t)
}

func (r *runner) wait(ctx context.Context, t *api.Transcription) (*api.Transcription, error) {
	announced := false
	var asked time.Time
	for {
		if shouldAnnounce(announced, t) {
			announced = true
			r.send(Transcribing)
		}

		if t.Terminal() {
			return t, nil
		}

		if err := r.pace(ctx, asked); err != nil {
			return t, err
		}

		asked = time.Now()
		next, err := r.c.Get(ctx, t.ID, waitSeconds)
		if err != nil {
			return t, err
		}

		if t.Reused {
			next.Reused = true
		}

		t = next
	}
}

// some servers answer wait=60 at once, avoid a hot loop
func (r *runner) pace(ctx context.Context, asked time.Time) error {
	if asked.IsZero() {
		return nil
	}

	pause := minPollGap - time.Since(asked)
	if pause <= 0 {
		return nil
	}

	return r.c.Sleep(ctx, pause)
}

func isWorking(t *api.Transcription) bool {
	return slices.Contains([]string{api.StatusFetching, api.StatusTranscribing}, t.Status)
}

func buildJobError(t *api.Transcription) error {
	apiErr := &api.Error{
		Code:    t.Status,
		Message: "The transcription was " + t.Status,
	}
	if t.Error != nil {
		apiErr.Code = t.Error.Code
		apiErr.Message = t.Error.Message
	}

	return apiErr
}
