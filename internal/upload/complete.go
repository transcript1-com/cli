package upload

import (
	"context"
	"errors"
	"fmt"

	"github.com/transcript1/cli/internal/api"
)

const errUnplannedChunk = "the server reported chunk %v missing, which is not in the upload plan: %w"

func (u *uploader) complete(ctx context.Context) (*api.Transcription, error) {
	t, err := u.c.CompleteUpload(ctx, u.id)
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

	return u.c.CompleteUpload(ctx, u.id)
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
