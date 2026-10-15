package upload

import (
	"io"
	"sync"

	"github.com/transcript1/cli/internal/api"
	"github.com/transcript1/cli/internal/inputs"
)

type uploader struct {
	c       *api.Client
	file    io.ReaderAt
	in      inputs.Input
	opt     Options
	id      string
	plan    *api.UploadPlan
	byIndex map[int]api.Chunk

	mu       sync.Mutex
	token    string
	gen      int
	renewing chan struct{}
	received map[int]bool
	counted  map[int]bool
	sent     int64
}

func newUploader(c *api.Client, file io.ReaderAt, in inputs.Input, opt Options, t *api.Transcription) *uploader {
	byIndex := map[int]api.Chunk{}
	for _, chunk := range t.Upload.Chunks {
		byIndex[chunk.Index] = chunk
	}

	return &uploader{
		c:        c,
		file:     file,
		in:       in,
		opt:      opt,
		id:       t.ID,
		plan:     t.Upload,
		byIndex:  byIndex,
		token:    t.Upload.Headers["X-Upload-Token"],
		received: map[int]bool{},
		counted:  map[int]bool{},
	}
}

func (u *uploader) getToken() (string, int) {
	u.mu.Lock()
	defer u.mu.Unlock()

	return u.token, u.gen
}

func (u *uploader) has(index int) bool {
	u.mu.Lock()
	defer u.mu.Unlock()

	return u.received[index]
}

func (u *uploader) acknowledge(index int) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.markReceived(index)
}

func (u *uploader) markReceived(index int) {
	chunk, ok := u.byIndex[index]
	if !ok {
		return
	}

	u.received[index] = true
	if u.counted[index] {
		return
	}

	u.counted[index] = true
	u.sent += chunk.Size
	if u.opt.Progress != nil {
		u.opt.Progress(u.sent, u.in.Size)
	}
}

func (u *uploader) forget(indexes []int) {
	u.mu.Lock()
	defer u.mu.Unlock()
	for _, index := range indexes {
		delete(u.received, index)
	}
}
