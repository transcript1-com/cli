package upload

import (
	"context"
	"errors"
	"os"

	"github.com/transcript1/cli/internal/api"
	"github.com/transcript1/cli/internal/inputs"
)

const (
	defaultParallel = 4
	maxRenewals     = 2
)

type Options struct {
	Create         api.CreateOptions
	IdempotencyKey string
	Parallel       int
	Created        func(id string)
	Progress       func(sent, total int64)
	WaitSlot       func(ctx context.Context, err *api.Error) error
}

func Run(ctx context.Context, c *api.Client, in inputs.Input, opt Options) (*api.Transcription, error) {
	file, err := os.Open(in.Path)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	t, err := c.CreateUpload(ctx, opt.IdempotencyKey, in.Name, in.Size, inputs.MimeType(in.Name), opt.Create)
	if err != nil {
		return nil, err
	}

	if opt.Created != nil {
		opt.Created(t.ID)
	}

	if t.Status != api.StatusAwaitingUpload {
		return t, nil
	}

	if t.Upload == nil {
		return nil, errors.New("the server sent no upload plan")
	}

	u := newUploader(c, file, in, opt, t)
	if err := u.send(ctx, t.Upload.Chunks); err != nil {
		return nil, err
	}

	return u.complete(ctx)
}
