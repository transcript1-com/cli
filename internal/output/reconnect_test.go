package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/transcript1-com/cli/internal/api"
	"github.com/transcript1-com/cli/internal/batch"
)

func buildRunning() *api.Transcription {
	return &api.Transcription{
		ID:     "tr_9",
		Status: api.StatusTranscribing,
		Raw:    json.RawMessage(`{"id":"tr_9","status":"transcribing"}`),
	}
}

func TestReconnecting(t *testing.T) {
	in := buildFileInput("a.mp3")

	t.Run("plain prints reconnecting, then the stage again", func(t *testing.T) {
		var buf bytes.Buffer
		r := NewReporter(&buf, Plain, 1, false)
		r.Event(batch.Event{Input: in, Stage: batch.Transcribing})
		r.Event(batch.Event{Input: in, Stage: batch.Reconnecting})
		r.Event(batch.Event{Input: in, Stage: batch.Transcribing})
		want := "[1/1] a.mp3: transcribing\n[1/1] a.mp3: reconnecting\n[1/1] a.mp3: transcribing\n"
		if buf.String() != want {
			t.Fatalf("output = %q, want %q", buf.String(), want)
		}
	})

	t.Run("a terminal shows reconnecting on the live line", func(t *testing.T) {
		var buf bytes.Buffer
		r := NewReporter(&buf, TTY, 1, false)
		r.Event(batch.Event{Input: in, Stage: batch.Reconnecting})
		if !strings.Contains(buf.String(), "[1/1] a.mp3  reconnecting") {
			t.Fatalf("output = %q", buf.String())
		}
	})
}

func TestGetHintOnFailure(t *testing.T) {
	in := buildFileInput("a.mp3")
	res := batch.Result{
		Status: batch.Failed,
		Err:    errors.New("boom"),
		T:      buildRunning(),
	}
	want := "transcript1: a.mp3: boom; fetch with transcript1 get tr_9\n"

	for _, mode := range []Mode{Plain, Quiet} {
		var buf bytes.Buffer
		NewReporter(&buf, mode, 1, false).Event(buildFinishEvent(0, in, res))
		if buf.String() != want {
			t.Fatalf("mode %v output = %q, want %q", mode, buf.String(), want)
		}
	}

	t.Run("a terminal shows it too", func(t *testing.T) {
		var buf bytes.Buffer
		NewReporter(&buf, TTY, 1, false).Event(buildFinishEvent(0, in, res))
		if !strings.Contains(buf.String(), strings.TrimSuffix(want, "\n")) {
			t.Fatalf("output = %q", buf.String())
		}
	})

	t.Run("the json line carries the id and the error", func(t *testing.T) {
		line, err := JSONLine(res)
		if err != nil {
			t.Fatal(err)
		}

		obj := decode(t, line)
		if string(obj["id"]) != `"tr_9"` {
			t.Fatalf("line %s", line)
		}

		if !strings.Contains(string(obj["error"]), "boom") {
			t.Fatalf("line %s", line)
		}
	})
}
