package batch

import (
	"context"
	"slices"
	"time"

	"github.com/transcript1-com/cli/internal/api"
)

const (
	minPollGap   = 2 * time.Second
	firstBackoff = 2 * time.Second
	maxBackoff   = 30 * time.Second
)

func shouldAnnounce(announced bool, t *api.Transcription) bool {
	if announced {
		return false
	}

	return isWorking(t)
}

func (r *runner) wait(ctx context.Context, t *api.Transcription) (*api.Transcription, error) {
	announced := false
	outage := 0
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
			outage, err = r.reconnect(ctx, err, outage)
			if err != nil {
				return t, err
			}

			asked = time.Time{}
			continue
		}

		if t.Reused {
			next.Reused = true
		}

		t = next
		if outage > 0 {
			outage = 0
			announced = r.resume(t)
		}
	}
}

// once the job exists, network errors only delay the answer
func (r *runner) reconnect(ctx context.Context, err error, outage int) (int, error) {
	if ctx.Err() != nil {
		return outage, err
	}

	if !api.Transient(err) {
		return outage, err
	}

	if outage == 0 {
		r.send(Reconnecting)
	}

	if err := r.c.Sleep(ctx, getBackoff(outage)); err != nil {
		return outage, err
	}

	return outage + 1, nil
}

func getBackoff(outage int) time.Duration {
	return min(firstBackoff<<min(outage, 4), maxBackoff)
}

func (r *runner) resume(t *api.Transcription) bool {
	if isWorking(t) {
		r.send(Transcribing)
		return true
	}

	if !t.Terminal() {
		r.send(Queued)
	}

	return false
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
