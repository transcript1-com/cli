package batch

import (
	"context"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/transcript1-com/cli/internal/api"
)

func recordSleeps(c *api.Client) *[]time.Duration {
	var slept []time.Duration
	c.Sleep = func(_ context.Context, d time.Duration) error {
		slept = append(slept, d)
		return nil
	}

	return &slept
}

func isBackoff(d time.Duration) bool {
	if d <= time.Second {
		return false
	}

	return d%time.Second == 0
}

func listBackoffs(slept []time.Duration) []time.Duration {
	var list []time.Duration
	for _, d := range slept {
		if isBackoff(d) {
			list = append(list, d)
		}
	}

	return list
}

func TestKeepsWaitingThroughServerErrors(t *testing.T) {
	srv, c := setup(t)
	srv.Polls = 2
	srv.PollErrors = 9
	slept := recordSleeps(c)
	var ev events
	res := runOne(context.Background(), c, 0, buildLink(rickID), Options{}, false, ev.emit)

	t.Run("finishes the input", func(t *testing.T) {
		if res.Status != Done {
			t.Fatalf("result %+v", res)
		}
	})

	t.Run("backs off 2, 4 and 8 seconds between failed polls", func(t *testing.T) {
		want := []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second}
		if got := listBackoffs(*slept); !slices.Equal(got, want) {
			t.Fatalf("backoffs %v (all sleeps %v)", got, *slept)
		}
	})

	t.Run("announces one reconnect, then goes back to transcribing", func(t *testing.T) {
		want := []Stage{Queued, Reconnecting, Transcribing, Finished}
		if got := ev.listStages(); !slices.Equal(got, want) {
			t.Fatalf("stages %v", got)
		}
	})
}

func TestKeepsWaitingThroughDroppedConnections(t *testing.T) {
	srv, c := setup(t)
	srv.PollErrors = 6
	srv.PollDrop = true
	var ev events
	res := runOne(context.Background(), c, 0, buildLink(rickID), Options{}, false, ev.emit)
	if res.Status != Done {
		t.Fatalf("result %+v stages %v", res, ev.listStages())
	}

	if !ev.has(Reconnecting) {
		t.Fatalf("result %+v stages %v", res, ev.listStages())
	}
}

func TestBackoffIsCappedAt30Seconds(t *testing.T) {
	srv, c := setup(t)
	srv.PollErrors = 3 * 7
	slept := recordSleeps(c)
	res := run(t, c, buildLink(rickID), Options{})
	want := []time.Duration{2, 4, 8, 16, 30, 30, 30}
	for i := range want {
		want[i] *= time.Second
	}

	got := listBackoffs(*slept)
	if res.Status != Done {
		t.Fatalf("result %+v backoffs %v", res, got)
	}

	if !slices.Equal(got, want) {
		t.Fatalf("result %+v backoffs %v", res, got)
	}
}

func TestStopsReconnectingWhenCanceled(t *testing.T) {
	srv, c := setup(t)
	srv.PollErrors = 100
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c.Sleep = func(ctx context.Context, d time.Duration) error {
		if d > time.Second {
			cancel()
		}

		return ctx.Err()
	}

	res := runOne(ctx, c, 0, buildLink(rickID), Options{}, false, nil)
	if res.Status != StillRunning {
		t.Fatalf("result %+v", res)
	}

	if res.T == nil {
		t.Fatalf("result %+v", res)
	}

	if res.T.ID != "tr_1" {
		t.Fatalf("result %+v", res)
	}
}

func TestFailsOnANotFoundWhileWaiting(t *testing.T) {
	srv, c := setup(t)
	srv.PollErrors = 1
	srv.PollErrorStatus = http.StatusNotFound
	var ev events
	res := runOne(context.Background(), c, 0, buildLink(rickID), Options{}, false, ev.emit)
	if res.Status != Failed {
		t.Fatalf("result %+v stages %v", res, ev.listStages())
	}

	if !api.IsCode(res.Err, api.CodeNotFound) {
		t.Fatalf("result %+v stages %v", res, ev.listStages())
	}

	if ev.has(Reconnecting) {
		t.Fatalf("result %+v stages %v", res, ev.listStages())
	}

	if res.GetHint() != "fetch with transcript1 get tr_1" {
		t.Fatalf("hint %q", res.GetHint())
	}
}
