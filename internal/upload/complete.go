package upload

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/transcript1-com/cli/internal/api"
)

const errUnplannedChunk = "the server reported chunk %v missing, which is not in the upload plan: %w"

func (u *uploader) complete(ctx context.Context) (*api.Transcription, error) {
	t, err := u.completeOnce(ctx)
	if !api.IsCode(err, api.CodeUploadIncomplete) {
		return t, err
	}

	chunks, err := u.findMissing(err)
	if err != nil {
		return nil, err
	}

	indexes := make([]int, len(chunks))
	for i, chunk := range chunks {
		indexes[i] = chunk.Index
	}

	u.forget(indexes)
	if err := u.send(ctx, chunks); err != nil {
		return nil, err
	}

	return u.completeOnce(ctx)
}

// the first answer may be lost, so the job may already exist
func (u *uploader) findStarted(ctx context.Context, err error) (*api.Transcription, error) {
	t, getErr := u.c.Get(ctx, u.id, 0)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	if getErr != nil {
		return nil, err
	}

	started := []string{api.StatusQueued, api.StatusFetching, api.StatusTranscribing, api.StatusCompleted}
	if !slices.Contains(started, t.Status) {
		return nil, err
	}

	return t, nil
}

func (u *uploader) completeOnce(ctx context.Context) (*api.Transcription, error) {
	for {
		t, err := u.c.CompleteUpload(ctx, u.id)
		if api.IsCode(err, api.CodeInvalidState) {
			return u.findStarted(ctx, err)
		}

		busy, ok := api.SlotBusy(err)
		if !ok {
			return t, err
		}

		if u.opt.WaitSlot == nil {
			return t, err
		}

		if err := u.opt.WaitSlot(ctx, busy); err != nil {
			return nil, err
		}
	}
}

func (u *uploader) findMissing(err error) ([]api.Chunk, error) {
	var apiErr *api.Error
	errors.As(err, &apiErr)
	raw, _ := apiErr.Details["missing_chunks"].([]any)
	if len(raw) == 0 {
		return nil, err
	}

	chunks := make([]api.Chunk, 0, len(raw))
	for _, value := range raw {
		index, ok := value.(float64)
		if !ok {
			return nil, fmt.Errorf(errUnplannedChunk, value, err)
		}

		chunk, planned := u.byIndex[int(index)]
		if !planned {
			return nil, fmt.Errorf(errUnplannedChunk, value, err)
		}

		chunks = append(chunks, chunk)
	}

	return chunks, nil
}
