package batch

import (
	"context"
	"slices"

	"github.com/transcript1/cli/internal/api"
)

func shouldAnnounce(announced bool, t *api.Transcription) bool {
	if announced {
		return false
	}

	return isWorking(t)
}

func (r *runner) wait(ctx context.Context, t *api.Transcription) (*api.Transcription, error) {
	announced := false
	for {
		if shouldAnnounce(announced, t) {
			announced = true
			r.send(Transcribing)
		}

		if t.Terminal() {
			return t, nil
		}

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
