package output

import (
	"bytes"
	"testing"

	"github.com/transcript1-com/cli/internal/api"
	"github.com/transcript1-com/cli/internal/batch"
)

func buildSummary(mode Mode, results []batch.Result) string {
	var buf bytes.Buffer
	NewReporter(&buf, mode, len(results), false).Summary(results)

	return buf.String()
}

func buildDone(n int) batch.Result {
	return batch.Result{Status: batch.Done, T: buildBilled(n)}
}

func TestSummary(t *testing.T) {
	t.Run("counts every outcome and the minutes used", func(t *testing.T) {
		results := []batch.Result{
			buildDone(10), buildDone(20), buildDone(12), buildDone(10),
			{Status: batch.Failed},
			{Status: batch.Skipped},
		}
		want := "4 done, 1 failed, 1 skipped · 52 minutes used\n"
		if got := buildSummary(Plain, results); got != want {
			t.Fatalf("Summary = %q, want %q", got, want)
		}
	})

	t.Run("leaves out zero counts other than done", func(t *testing.T) {
		if got := buildSummary(Plain, []batch.Result{buildDone(3)}); got != "1 done · 3 minutes used\n" {
			t.Fatalf("Summary = %q", got)
		}
	})

	t.Run("a reused transcription adds no minutes", func(t *testing.T) {
		reused := buildDone(9)
		reused.T.Reused = true
		if got := buildSummary(Plain, []batch.Result{reused, buildDone(1)}); got != "2 done · 1 minute used\n" {
			t.Fatalf("Summary = %q", got)
		}
	})

	t.Run("started counts as done and still running is its own part", func(t *testing.T) {
		results := []batch.Result{
			{Status: batch.Started, T: &api.Transcription{ID: "tr_1"}},
			{Status: batch.StillRunning, T: &api.Transcription{ID: "tr_2"}},
		}
		want := "1 done, 1 still running · 0 minutes used\n"
		if got := buildSummary(Plain, results); got != want {
			t.Fatalf("Summary = %q, want %q", got, want)
		}
	})

	t.Run("interrupted counts as failed and out of minutes as skipped", func(t *testing.T) {
		results := []batch.Result{buildDone(1), {Status: batch.Interrupted}, {Status: batch.SkippedMinutes}}
		want := "1 done, 1 failed, 1 skipped · 1 minute used\n"
		if got := buildSummary(TTY, results); got != want {
			t.Fatalf("Summary = %q, want %q", got, want)
		}
	})

	t.Run("a done result without billing data adds nothing", func(t *testing.T) {
		results := []batch.Result{{Status: batch.Done, T: &api.Transcription{ID: "tr_1"}}}
		if got := buildSummary(Plain, results); got != "1 done · 0 minutes used\n" {
			t.Fatalf("Summary = %q", got)
		}
	})
}
