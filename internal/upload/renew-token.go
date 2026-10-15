package upload

import (
	"context"

	"github.com/transcript1/cli/internal/api"
	"github.com/transcript1/cli/internal/inputs"
)

// first worker to get a 401 renews, the others wait
func (u *uploader) renew(ctx context.Context, gen int) (bool, error) {
	u.mu.Lock()
	if u.gen != gen {
		u.mu.Unlock()
		return false, nil
	}

	if wait := u.renewing; wait != nil {
		u.mu.Unlock()
		select {
		case <-wait:
			return false, nil
		case <-ctx.Done():
			return false, ctx.Err()
		}
	}

	done := make(chan struct{})
	u.renewing = done
	u.mu.Unlock()

	token, received, err := u.fetchToken(ctx)
	u.mu.Lock()
	defer u.mu.Unlock()
	u.renewing = nil
	close(done)
	if err != nil {
		return false, err
	}

	u.token = token
	u.gen++
	for _, index := range received {
		u.markReceived(index)
	}

	return true, nil
}

func buildStateError(status string) *api.Error {
	return &api.Error{
		Code:    api.CodeInvalidState,
		Message: "the transcription is " + status + ", not awaiting an upload",
	}
}

// repeating the create with the same key rotates the token
func (u *uploader) fetchToken(ctx context.Context) (string, []int, error) {
	t, err := u.c.CreateUpload(ctx, u.opt.IdempotencyKey, u.in.Name, u.in.Size, inputs.MimeType(u.in.Name), u.opt.Create)
	if err != nil {
		return "", nil, err
	}

	if t.Status != api.StatusAwaitingUpload {
		return "", nil, buildStateError(t.Status)
	}

	if t.Upload == nil {
		return "", nil, buildStateError(t.Status)
	}

	progress, err := u.c.UploadProgress(ctx, u.id)
	if err != nil {
		return "", nil, err
	}

	return t.Upload.Headers["X-Upload-Token"], progress.ReceivedChunks, nil
}
