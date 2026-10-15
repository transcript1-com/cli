package batch

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

type failDelete struct {
	next http.RoundTripper
}

func (f failDelete) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.Method == http.MethodDelete {
		return nil, errors.New("network down")
	}

	return f.next.RoundTrip(r)
}

func interruptUpload(t *testing.T, failCleanup bool) (Result, []string) {
	t.Helper()
	srv, c := setup(t)
	if failCleanup {
		c.HTTP.Transport = failDelete{next: c.HTTP.Transport}
	}

	srv.BlockChunks = make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		for srv.PeakChunks() == 0 {
			time.Sleep(time.Millisecond)
		}

		cancel()
	}()

	res := runOne(ctx, c, 0, buildFileInput(t, "talk.mp3"), Options{}, false, nil)

	return res, srv.Log()
}

func TestCancelUpload(t *testing.T) {
	t.Run("deletes the upload left awaiting chunks", func(t *testing.T) {
		res, log := interruptUpload(t, false)
		if res.Status != Interrupted {
			t.Fatalf("result %+v log %v", res, log)
		}

		if res.Err != nil {
			t.Fatalf("result %+v log %v", res, log)
		}

		if log[len(log)-1] != "DELETE /v1/transcriptions/tr_1" {
			t.Fatalf("result %+v log %v", res, log)
		}
	})

	t.Run("keeps the delete error when the cleanup fails", func(t *testing.T) {
		res, _ := interruptUpload(t, true)
		if res.Status != Interrupted {
			t.Fatalf("result %+v", res)
		}

		if res.Err == nil {
			t.Fatalf("result %+v", res)
		}

		if !strings.Contains(res.Err.Error(), "tr_1") {
			t.Fatalf("result %+v", res)
		}
	})

	t.Run("deletes nothing for a link canceled while waiting", func(t *testing.T) {
		srv, c := setup(t)
		srv.Polls = 100
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		emit := func(ev Event) {
			if ev.Stage == Transcribing {
				cancel()
			}
		}

		res := runOne(ctx, c, 0, buildLink(rickID), Options{}, false, emit)
		for _, line := range srv.Log() {
			if strings.HasPrefix(line, "DELETE ") {
				t.Fatalf("result %+v log %v", res, srv.Log())
			}
		}
	})
}
