package upload

import (
	"context"
	"io"
	"sync"

	"github.com/transcript1/cli/internal/api"
)

func (u *uploader) send(parent context.Context, chunks []api.Chunk) error {
	ctx, cancel := context.WithCancelCause(parent)
	defer cancel(nil)
	queue := make(chan api.Chunk)
	var wg sync.WaitGroup
	for range min(u.getParallel(), len(chunks)) {
		wg.Go(func() {
			for chunk := range queue {
				if err := u.sendChunk(ctx, chunk); err != nil {
					cancel(err)
					return
				}
			}
		})
	}

feed:
	for _, chunk := range chunks {
		select {
		case queue <- chunk:
		case <-ctx.Done():
			break feed
		}
	}

	close(queue)
	wg.Wait()
	if err := parent.Err(); err != nil {
		return err
	}

	return context.Cause(ctx)
}

func (u *uploader) getParallel() int {
	if u.opt.Parallel > 0 {
		return u.opt.Parallel
	}

	return defaultParallel
}

func (u *uploader) sendChunk(ctx context.Context, chunk api.Chunk) error {
	chunkURL, err := u.c.ChunkURL(u.plan, chunk.Index)
	if err != nil {
		return err
	}

	renewals := 0
	for !u.has(chunk.Index) {
		token, gen := u.getToken()
		err := u.c.PutChunk(ctx, chunkURL, token, io.NewSectionReader(u.file, chunk.Offset, chunk.Size))
		if err == nil {
			u.acknowledge(chunk.Index)
			return nil
		}

		if !api.IsCode(err, api.CodeUnauthorized) {
			return err
		}

		if renewals == maxRenewals {
			return err
		}

		renewed, err := u.renew(ctx, gen)
		if err != nil {
			return err
		}

		if renewed {
			renewals++
		}
	}

	return nil
}
