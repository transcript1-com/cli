package batch

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/transcript1/cli/internal/api"
	"github.com/transcript1/cli/internal/fakeapi"
	"github.com/transcript1/cli/internal/inputs"
)

func buildLinks(n int) []inputs.Input {
	ins := make([]inputs.Input, n)
	for i := range ins {
		ins[i] = buildLink(fmt.Sprintf("video%06d", i))
	}

	return ins
}

func countPosts(log []string) int {
	n := 0
	for _, line := range log {
		if line == postPath {
			n++
		}
	}

	return n
}

func (e *events) listFinished(idx int) []Event {
	e.mu.Lock()
	defer e.mu.Unlock()
	var list []Event
	for _, ev := range e.list {
		if ev.Index != idx {
			continue
		}

		if ev.Stage == Finished {
			list = append(list, ev)
		}
	}

	return list
}

func (e *events) countForIndex(idx int) int {
	e.mu.Lock()
	defer e.mu.Unlock()
	n := 0
	for _, ev := range e.list {
		if ev.Index == idx {
			n++
		}
	}

	return n
}

func TestOrderAndConcurrency(t *testing.T) {
	srv, c := setup(t)
	srv.Polls = 3
	ins := buildLinks(6)
	var ev events
	results, err := Run(context.Background(), c, ins, Options{Jobs: 3}, ev.emit)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("returns results in input order", func(t *testing.T) {
		if len(results) != len(ins) {
			t.Fatalf("%d results", len(results))
		}

		for i, res := range results {
			if res.Index != i {
				t.Fatalf("result %d: %+v", i, res)
			}

			if res.Input.VideoID != ins[i].VideoID {
				t.Fatalf("result %d: %+v", i, res)
			}

			if res.Status != Done {
				t.Fatalf("result %d: %+v", i, res)
			}
		}
	})

	t.Run("never runs more jobs than Jobs", func(t *testing.T) {
		peak := srv.PeakActive()
		if peak > 3 {
			t.Fatalf("peak %d", peak)
		}

		if peak < 1 {
			t.Fatalf("peak %d", peak)
		}
	})

	t.Run("emits one finished event per input", func(t *testing.T) {
		for i := range ins {
			if n := len(ev.listFinished(i)); n != 1 {
				t.Fatalf("input %d finished %d times", i, n)
			}
		}
	})
}

func TestOutOfMinutes(t *testing.T) {
	srv, c := setup(t)
	srv.MinutesLeft = 2
	var ev events
	results, err := Run(context.Background(), c, buildLinks(4), Options{Jobs: 1}, ev.emit)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("finishes the paid inputs and stops at the first refusal", func(t *testing.T) {
		got := []Status{results[0].Status, results[1].Status, results[2].Status, results[3].Status}
		if want := []Status{Done, Done, Failed, SkippedMinutes}; !slices.Equal(got, want) {
			t.Fatalf("statuses %v", got)
		}

		if !api.IsCode(results[2].Err, api.CodeInsufficientMinutes) {
			t.Fatalf("err %v", results[2].Err)
		}
	})

	t.Run("sends nothing for the skipped input", func(t *testing.T) {
		if n := countPosts(srv.Log()); n != 3 {
			t.Fatalf("%d creates: %v", n, srv.Log())
		}
	})

	t.Run("reports the skipped input as finished", func(t *testing.T) {
		list := ev.listFinished(3)
		if len(list) != 1 {
			t.Fatalf("events %+v", list)
		}

		if list[0].Result.Status != SkippedMinutes {
			t.Fatalf("events %+v", list)
		}
	})
}

func TestUnauthorizedStops(t *testing.T) {
	srv, _ := setup(t)
	c := api.New(srv.URL, "t1_wrong")
	c.Sleep = func(context.Context, time.Duration) error { return nil }
	var ev events
	results, err := Run(context.Background(), c, buildLinks(8), Options{Jobs: 2}, ev.emit)

	t.Run("returns the unauthorized error", func(t *testing.T) {
		if !api.IsCode(err, api.CodeUnauthorized) {
			t.Fatalf("err %v", err)
		}
	})

	t.Run("sends at most one create per worker", func(t *testing.T) {
		n := countPosts(srv.Log())
		if n > 2 {
			t.Fatalf("%d creates: %v", n, srv.Log())
		}

		if n < 1 {
			t.Fatalf("%d creates: %v", n, srv.Log())
		}
	})

	t.Run("leaves the rest not started and silent", func(t *testing.T) {
		if len(results) != 8 {
			t.Fatalf("%d results", len(results))
		}

		for i := 2; i < 8; i++ {
			if results[i].Status != NotStarted {
				t.Fatalf("result %d: %+v, %d events", i, results[i], ev.countForIndex(i))
			}

			if results[i].Index != i {
				t.Fatalf("result %d: %+v, %d events", i, results[i], ev.countForIndex(i))
			}

			if ev.countForIndex(i) != 0 {
				t.Fatalf("result %d: %+v, %d events", i, results[i], ev.countForIndex(i))
			}
		}
	})
}

func waitForChunk(srv *fakeapi.Server) {
	deadline := time.Now().Add(5 * time.Second)
	for srv.PeakChunks() == 0 {
		if !time.Now().Before(deadline) {
			return
		}

		time.Sleep(time.Millisecond)
	}
}

func TestInterrupt(t *testing.T) {
	srv, c := setup(t)
	srv.Polls = 100
	srv.BlockChunks = make(chan struct{})
	ins := []inputs.Input{buildFileInput(t, "talk.mp3"), buildLink(rickID), buildLink("aaaaaaaaaaa")}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var ev events
	emit := func(e Event) {
		ev.emit(e)
		if e.Index != 1 {
			return
		}

		if e.Stage != Transcribing {
			return
		}

		waitForChunk(srv)
		cancel()
	}

	results, err := Run(ctx, c, ins, Options{Jobs: 2}, emit)
	if err != nil {
		t.Fatal(err)
	}

	log := srv.Log()

	t.Run("cancels the unfinished upload", func(t *testing.T) {
		res := results[0]
		if res.Status != Interrupted {
			t.Fatalf("result %+v", res)
		}

		if res.T == nil {
			t.Fatalf("result %+v", res)
		}

		if res.T.Status != api.StatusAwaitingUpload {
			t.Fatalf("result %+v", res)
		}

		if res.Err != nil {
			t.Fatalf("result %+v", res)
		}

		if !slices.Contains(log, "DELETE /v1/transcriptions/"+res.T.ID) {
			t.Fatalf("log %v", log)
		}
	})

	t.Run("leaves the processing link running", func(t *testing.T) {
		res := results[1]
		if res.Status != StillRunning {
			t.Fatalf("result %+v", res)
		}

		if res.T == nil {
			t.Fatalf("result %+v", res)
		}

		if res.T.ID == "" {
			t.Fatalf("result %+v", res)
		}

		if slices.Contains(log, "DELETE /v1/transcriptions/"+res.T.ID) {
			t.Fatalf("log %v", log)
		}
	})

	t.Run("does not start the waiting input", func(t *testing.T) {
		if results[2].Status != NotStarted {
			t.Fatalf("result %+v, %d events", results[2], ev.countForIndex(2))
		}

		if ev.countForIndex(2) != 0 {
			t.Fatalf("result %+v, %d events", results[2], ev.countForIndex(2))
		}

		if n := countPosts(log); n != 2 {
			t.Fatalf("%d creates: %v", n, log)
		}
	})

	t.Run("sends one delete", func(t *testing.T) {
		n := 0
		for _, line := range log {
			if strings.HasPrefix(line, "DELETE ") {
				n++
			}
		}

		if n != 1 {
			t.Fatalf("log %v", log)
		}
	})
}
