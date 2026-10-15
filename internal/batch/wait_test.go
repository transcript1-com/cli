package batch

import (
	"context"
	"testing"
	"time"
)

func TestPollsAnsweredAtOnceAreSpaced(t *testing.T) {
	srv, c := setup(t)
	srv.Polls = 4
	var slept []time.Duration
	c.Sleep = func(_ context.Context, d time.Duration) error {
		slept = append(slept, d)
		return nil
	}

	res := run(t, c, buildLink(rickID), Options{})

	t.Run("finishes the input", func(t *testing.T) {
		if res.Status != Done {
			t.Fatalf("result %+v", res)
		}
	})

	t.Run("sleeps about 2 seconds before each poll that follows a non-terminal answer", func(t *testing.T) {
		if len(slept) != 3 {
			t.Fatalf("slept %v", slept)
		}

		for _, d := range slept {
			if d < 1900*time.Millisecond {
				t.Fatalf("slept %v", slept)
			}

			if d > 2*time.Second {
				t.Fatalf("slept %v", slept)
			}
		}
	})

	t.Run("does not sleep after the terminal answer", func(t *testing.T) {
		if got := srv.Log(); len(got) != 6 {
			t.Fatalf("log %v", got)
		}
	})
}
