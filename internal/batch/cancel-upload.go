package batch

import (
	"context"
	"fmt"
	"time"

	"github.com/transcript1-com/cli/internal/api"
)

const (
	cancelUploadTimeout = 5 * time.Second
	errCancelUpload     = "could not cancel the unfinished upload %s: %w"
)

func isUnfinishedUpload(res Result) bool {
	if res.Status != Interrupted {
		return false
	}

	if res.T == nil {
		return false
	}

	return res.T.Status == api.StatusAwaitingUpload
}

// the run context is already canceled, so use a fresh one
func (r *runner) cancelUpload(res Result) Result {
	if !isUnfinishedUpload(res) {
		return res
	}

	ctx, cancel := context.WithTimeout(context.Background(), cancelUploadTimeout)
	defer cancel()
	t, err := r.c.Get(ctx, res.T.ID, 0)
	if err != nil {
		res.Err = fmt.Errorf(errCancelUpload, res.T.ID, err)
		return res
	}

	if t.Status != api.StatusAwaitingUpload {
		res.Status = StillRunning
		res.T = t

		return res
	}

	if err := r.c.Delete(ctx, res.T.ID); err != nil {
		res.Err = fmt.Errorf(errCancelUpload, res.T.ID, err)
	}

	return res
}
