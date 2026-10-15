package batch

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/transcript1/cli/internal/api"
)

func TestJobFailed(t *testing.T) {
	srv, c := setup(t)
	srv.FailCode = api.CodeMediaPrivate
	res := run(t, c, buildLink(rickID), Options{})

	if res.Status != Failed {
		t.Fatalf("result %+v", res)
	}

	if !api.IsCode(res.Err, api.CodeMediaPrivate) {
		t.Fatalf("result %+v", res)
	}

	var apiErr *api.Error
	if !errors.As(res.Err, &apiErr) {
		t.Fatalf("err %#v", res.Err)
	}

	if apiErr.Status != 0 {
		t.Fatalf("err %#v", res.Err)
	}

	if apiErr.Message == "" {
		t.Fatalf("err %#v", res.Err)
	}

	if want := []string{postPath, getPath}; !slices.Equal(srv.Log(), want) {
		t.Fatalf("log %v", srv.Log())
	}
}

func TestNoWait(t *testing.T) {
	srv, c := setup(t)
	res := run(t, c, buildLink(rickID), Options{NoWait: true})
	if res.Status != Started {
		t.Fatalf("result %+v", res)
	}

	if res.T == nil {
		t.Fatalf("result %+v", res)
	}

	if res.T.ID == "" {
		t.Fatalf("result %+v", res)
	}

	if want := []string{postPath}; !slices.Equal(srv.Log(), want) {
		t.Fatalf("log %v", srv.Log())
	}
}

func TestWaitsForSlot(t *testing.T) {
	srv, c := setup(t)
	srv.ActiveLimit = 1
	held, err := c.CreateLink(context.Background(), "held", buildLink("aaaaaaaaaaa").URL, api.CreateOptions{}, 0, false)
	if err != nil {
		t.Fatal(err)
	}

	var slept []time.Duration
	c.Sleep = func(ctx context.Context, d time.Duration) error {
		slept = append(slept, d)
		_, err := c.Get(ctx, held.ID, 0)

		return err
	}

	var ev events
	res := runOne(context.Background(), c, 0, buildLink(rickID), Options{}, false, ev.emit)

	t.Run("announces the wait and sleeps for Retry-After", func(t *testing.T) {
		if !ev.has(WaitingSlot) {
			t.Fatalf("stages %v slept %v", ev.listStages(), slept)
		}

		if !slices.Equal(slept, []time.Duration{time.Second}) {
			t.Fatalf("stages %v slept %v", ev.listStages(), slept)
		}
	})

	t.Run("then finishes the input", func(t *testing.T) {
		if res.Status != Done {
			t.Fatalf("result %+v", res)
		}
	})

	t.Run("retries the create with the same idempotency key", func(t *testing.T) {
		var keys []string
		for _, req := range srv.Requests() {
			if req.Method != "POST" {
				continue
			}

			if req.Header.Get("Idempotency-Key") == "held" {
				continue
			}

			keys = append(keys, req.Header.Get("Idempotency-Key"))
		}

		if len(keys) != 2 {
			t.Fatalf("keys %v", keys)
		}

		if keys[0] != keys[1] {
			t.Fatalf("keys %v", keys)
		}

		if keys[0] == "" {
			t.Fatalf("keys %v", keys)
		}
	})
}

func TestFileWaitsForSlotAtComplete(t *testing.T) {
	srv, c := setup(t)
	srv.ActiveLimit = 1
	held, err := c.CreateLink(context.Background(), "held", buildLink("aaaaaaaaaaa").URL, api.CreateOptions{}, 0, false)
	if err != nil {
		t.Fatal(err)
	}

	var ev events
	emit := func(e Event) {
		ev.emit(e)
		if e.Stage == WaitingSlot {
			_, _ = c.Get(context.Background(), held.ID, 0)
		}
	}

	res := runOne(context.Background(), c, 0, buildFileInput(t, "talk.mp3"), Options{}, false, emit)

	if res.Status != Done {
		t.Fatalf("result %+v stages %v", res, ev.listStages())
	}

	if !ev.has(WaitingSlot) {
		t.Fatalf("result %+v stages %v", res, ev.listStages())
	}

	puts := map[int]int{}
	for _, req := range srv.Requests() {
		if req.Chunk >= 0 {
			puts[req.Chunk]++
		}
	}

	for index, n := range puts {
		if n != 1 {
			t.Fatalf("chunk %d sent %d times", index, n)
		}
	}

	n := countComplete(srv.Log())
	if n < 2 {
		t.Fatalf("%d completes, chunks %v", n, puts)
	}

	if len(puts) != 3 {
		t.Fatalf("%d completes, chunks %v", n, puts)
	}
}

func countComplete(log []string) int {
	n := 0
	for _, line := range log {
		if strings.HasSuffix(line, "/upload/complete") {
			n++
		}
	}

	return n
}

func TestInterrupted(t *testing.T) {
	t.Run("reports the created upload when canceled while sending chunks", func(t *testing.T) {
		srv, c := setup(t)
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
		if res.Status != Interrupted {
			t.Fatalf("result %+v", res)
		}

		if res.T == nil {
			t.Fatalf("result %+v", res)
		}

		if res.T.ID != "tr_1" {
			t.Fatalf("result %+v", res)
		}

		if res.T.Status != api.StatusAwaitingUpload {
			t.Fatalf("result %+v", res)
		}
	})

	t.Run("does not start after the context is canceled", func(t *testing.T) {
		srv, c := setup(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		res := runOne(ctx, c, 0, buildLink(rickID), Options{}, false, nil)
		if res.Status != NotStarted {
			t.Fatalf("result %+v log %v", res, srv.Log())
		}

		if len(srv.Log()) != 0 {
			t.Fatalf("result %+v log %v", res, srv.Log())
		}
	})

	t.Run("reports a job still running when canceled while waiting", func(t *testing.T) {
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
		if res.Status != StillRunning {
			t.Fatalf("result %+v", res)
		}

		if res.T == nil {
			t.Fatalf("result %+v", res)
		}

		if res.T.ID != "tr_1" {
			t.Fatalf("result %+v", res)
		}
	})
}
