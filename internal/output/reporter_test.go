package output

import (
	"bytes"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/transcript1-com/cli/internal/api"
	"github.com/transcript1-com/cli/internal/batch"
	"github.com/transcript1-com/cli/internal/inputs"
)

func buildFileInput(name string) inputs.Input {
	in := inputs.Input{
		Raw:  "talks/" + name,
		Kind: inputs.File,
		Name: name,
	}

	return in
}

func buildBilled(n int) *api.Transcription {
	return &api.Transcription{ID: "tr_1", BilledMinutes: &n}
}

func buildFinishEvent(index int, in inputs.Input, res batch.Result) batch.Event {
	res.Index = index
	res.Input = in
	e := batch.Event{
		Index:  index,
		Input:  in,
		Stage:  batch.Finished,
		Result: &res,
	}

	return e
}

func TestPlainProgress(t *testing.T) {
	var buf bytes.Buffer
	r := NewReporter(&buf, Plain, 5, false)
	in := buildFileInput("lecture.mp3")
	r.Event(batch.Event{Index: 1, Input: in, Stage: batch.Uploading, Sent: 10, Total: 100})
	r.Event(batch.Event{Index: 1, Input: in, Stage: batch.Uploading, Sent: 50, Total: 100})
	r.Event(batch.Event{Index: 1, Input: in, Stage: batch.Transcribing})
	r.Event(buildFinishEvent(1, in, batch.Result{Status: batch.Done, T: buildBilled(12), File: "transcripts/lecture.txt"}))
	want := "[2/5] lecture.mp3: uploading\n" +
		"[2/5] lecture.mp3: transcribing\n" +
		"[2/5] lecture.mp3: done · 12 min · transcripts/lecture.txt\n"
	if buf.String() != want {
		t.Fatalf("output = %q, want %q", buf.String(), want)
	}
}

func TestPlainStages(t *testing.T) {
	in := buildFileInput("a.mp3")
	t.Run("prints queued and slot waits once per change", func(t *testing.T) {
		var buf bytes.Buffer
		r := NewReporter(&buf, Plain, 1, false)
		r.Event(batch.Event{Input: in, Stage: batch.WaitingSlot})
		r.Event(batch.Event{Input: in, Stage: batch.WaitingSlot})
		r.Event(batch.Event{Input: in, Stage: batch.Queued})
		want := "[1/1] a.mp3: waiting for a free job slot\n[1/1] a.mp3: queued\n"
		if buf.String() != want {
			t.Fatalf("output = %q, want %q", buf.String(), want)
		}
	})

	t.Run("a done line without a file has no path", func(t *testing.T) {
		var buf bytes.Buffer
		r := NewReporter(&buf, Plain, 1, false)
		r.Event(buildFinishEvent(0, in, batch.Result{Status: batch.Done, T: buildBilled(3)}))
		if buf.String() != "[1/1] a.mp3: done · 3 min\n" {
			t.Fatalf("output = %q", buf.String())
		}
	})

	t.Run("a reused transcription shows reused", func(t *testing.T) {
		var buf bytes.Buffer
		r := NewReporter(&buf, Plain, 1, false)
		tr := buildBilled(0)
		tr.Reused = true
		r.Event(buildFinishEvent(0, in, batch.Result{Status: batch.Done, T: tr}))
		if buf.String() != "[1/1] a.mp3: done · reused\n" {
			t.Fatalf("output = %q", buf.String())
		}
	})

	t.Run("skipped inputs say why", func(t *testing.T) {
		var buf bytes.Buffer
		r := NewReporter(&buf, Plain, 2, false)
		r.Event(buildFinishEvent(0, in, batch.Result{Status: batch.Skipped, File: "out/a.txt"}))
		r.Event(buildFinishEvent(1, in, batch.Result{Status: batch.SkippedMinutes}))
		want := "[1/2] a.mp3: skipped (already exists: out/a.txt)\n[2/2] a.mp3: skipped (out of minutes)\n"
		if buf.String() != want {
			t.Fatalf("output = %q, want %q", buf.String(), want)
		}
	})

	t.Run("a failed input prints its error line", func(t *testing.T) {
		var buf bytes.Buffer
		r := NewReporter(&buf, Plain, 1, false)
		r.Event(buildFinishEvent(0, in, batch.Result{Status: batch.Failed, Err: errors.New("boom")}))
		if buf.String() != "transcript1: a.mp3: boom\n" {
			t.Fatalf("output = %q", buf.String())
		}
	})
}

func TestQuietPrintsOnlyErrors(t *testing.T) {
	var buf bytes.Buffer
	r := NewReporter(&buf, Quiet, 2, false)
	in := buildFileInput("a.mp3")
	r.Event(batch.Event{Input: in, Stage: batch.Uploading, Sent: 1, Total: 2})
	r.Event(batch.Event{Input: in, Stage: batch.Queued})
	r.Event(buildFinishEvent(0, in, batch.Result{Status: batch.Done, T: buildBilled(1)}))
	r.Event(buildFinishEvent(1, in, batch.Result{Status: batch.Failed, Err: errors.New("boom")}))
	r.Summary([]batch.Result{{Status: batch.Done, T: buildBilled(1)}})
	if buf.String() != "transcript1: a.mp3: boom\n" {
		t.Fatalf("output = %q", buf.String())
	}
}

func TestJSONModePrintsNothing(t *testing.T) {
	var buf bytes.Buffer
	r := NewReporter(&buf, JSON, 1, false)
	in := buildFileInput("a.mp3")
	r.Event(batch.Event{Input: in, Stage: batch.Queued})
	r.Event(buildFinishEvent(0, in, batch.Result{Status: batch.Failed, Err: errors.New("boom")}))
	r.Summary([]batch.Result{{Status: batch.Failed}})
	if buf.Len() != 0 {
		t.Fatalf("output = %q", buf.String())
	}
}

func TestReporterIsSafeForConcurrentEvents(t *testing.T) {
	var buf bytes.Buffer
	r := NewReporter(&buf, Plain, 20, false)
	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			in := buildFileInput("a.mp3")
			r.Event(batch.Event{Index: i, Input: in, Stage: batch.Queued})
			r.Event(buildFinishEvent(i, in, batch.Result{Status: batch.Done, T: buildBilled(1)}))
		}()
	}

	wg.Wait()
	if got := strings.Count(buf.String(), "\n"); got != 40 {
		t.Fatalf("lines = %d, want 40", got)
	}
}
