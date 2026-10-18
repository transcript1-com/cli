package upload

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"github.com/transcript1-com/cli/internal/api"
)

type badKeyAfterFirstCreate struct {
	next    http.RoundTripper
	mu      sync.Mutex
	creates int
}

func isCreate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	return r.URL.Path == "/v1/transcriptions"
}

func (b *badKeyAfterFirstCreate) RoundTrip(r *http.Request) (*http.Response, error) {
	if !isCreate(r) {
		return b.next.RoundTrip(r)
	}

	b.mu.Lock()
	b.creates++
	n := b.creates
	b.mu.Unlock()
	if n > 1 {
		r = r.Clone(r.Context())
		r.Header.Set("Authorization", "Bearer t1_wrong")
	}

	return b.next.RoundTrip(r)
}

func TestRenewalWithABadKey(t *testing.T) {
	srv, c, in, _ := setup(t, "talk.mp3")
	srv.ChunkStatus = map[int][]int{3: {401}}
	c.HTTP.Transport = &badKeyAfterFirstCreate{next: c.HTTP.Transport}
	_, err := Run(context.Background(), c, in, Options{IdempotencyKey: "idem-1"})

	t.Run("fails with unauthorized when the renewal create is refused", func(t *testing.T) {
		if !api.IsCode(err, api.CodeUnauthorized) {
			t.Fatalf("got %v", err)
		}

		if api.IsCode(err, api.CodeUploadTokenRefused) {
			t.Fatalf("got %v", err)
		}
	})
}
