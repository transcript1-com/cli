package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/transcript1-com/cli/internal/api"
	"github.com/transcript1-com/cli/internal/batch"
	"github.com/transcript1-com/cli/internal/inputs"
)

const finishedRaw = `{"id":"tr_1","status":"completed","billed_minutes":12,"media":{"title":"A <b> & B"},` +
	`"transcript":{"text":"hello"},"future_field":[1,2]}`

func buildLinkInput() inputs.Input {
	in := inputs.Input{
		Raw:  "https://youtu.be/abc",
		Kind: inputs.Link,
		URL:  "https://www.youtube.com/watch?v=abc",
	}

	return in
}

func decode(t *testing.T, line []byte) map[string]json.RawMessage {
	t.Helper()
	if !bytes.HasSuffix(line, []byte("\n")) {
		t.Fatalf("line %q is not one newline-terminated line", line)
	}

	if bytes.Count(line, []byte("\n")) != 1 {
		t.Fatalf("line %q is not one newline-terminated line", line)
	}

	if bytes.HasSuffix(bytes.TrimSuffix(line, []byte("\n")), []byte(" ")) {
		t.Fatalf("line %q has trailing spaces", line)
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(line, &obj); err != nil {
		t.Fatalf("line %q is not json: %v", line, err)
	}

	return obj
}

func buildFinishedResult(file string, reused bool) batch.Result {
	return batch.Result{
		Input:  buildLinkInput(),
		Status: batch.Done,
		File:   file,
		T: &api.Transcription{
			ID:     "tr_1",
			Reused: reused,
			Raw:    json.RawMessage(finishedRaw),
		},
	}
}

func TestJSONLine(t *testing.T) {
	t.Run("a finished job keeps the api object and adds input and reused", func(t *testing.T) {
		line, err := JSONLine(buildFinishedResult("", true))
		if err != nil {
			t.Fatal(err)
		}

		obj := decode(t, line)
		for key, want := range map[string]string{
			"id":           `"tr_1"`,
			"input":        `"https://youtu.be/abc"`,
			"reused":       "true",
			"future_field": "[1,2]",
			"transcript":   `{"text":"hello"}`,
			"media":        `{"title":"A <b> & B"}`,
		} {
			if string(obj[key]) != want {
				t.Fatalf("%s = %s, want %s (line %s)", key, obj[key], want, line)
			}
		}

		if _, ok := obj["file"]; ok {
			t.Fatalf("line %s has a file", line)
		}
	})

	t.Run("a written file adds file and drops the transcript", func(t *testing.T) {
		line, err := JSONLine(buildFinishedResult("out/a.txt", false))
		if err != nil {
			t.Fatal(err)
		}

		obj := decode(t, line)
		if string(obj["file"]) != `"out/a.txt"` {
			t.Fatalf("line = %s", line)
		}

		if string(obj["reused"]) != "false" {
			t.Fatalf("line = %s", line)
		}

		if _, ok := obj["transcript"]; ok {
			t.Fatalf("line %s still has the transcript", line)
		}

		if string(obj["billed_minutes"]) != "12" {
			t.Fatalf("line = %s", line)
		}
	})

	t.Run("an error before an id has input and error", func(t *testing.T) {
		res := batch.Result{
			Input:  buildLinkInput(),
			Status: batch.Failed,
			Err:    &api.Error{Code: api.CodeInvalidURL, Message: "Bad link"},
		}
		line, err := JSONLine(res)
		if err != nil {
			t.Fatal(err)
		}

		want := `{"error":{"code":"invalid_url","message":"Bad link"},"input":"https://youtu.be/abc"}` + "\n"
		if string(line) != want {
			t.Fatalf("line = %s, want %s", line, want)
		}
	})

	t.Run("a non-api error uses the code error", func(t *testing.T) {
		res := batch.Result{
			Input:  buildLinkInput(),
			Status: batch.Failed,
			Err:    errors.New("boom"),
		}
		line, err := JSONLine(res)
		if err != nil {
			t.Fatal(err)
		}

		want := `{"error":{"code":"error","message":"boom"},"input":"https://youtu.be/abc"}` + "\n"
		if string(line) != want {
			t.Fatalf("line = %s, want %s", line, want)
		}
	})
}

func TestJSONLineSkipped(t *testing.T) {
	t.Run("an existing output says exists with the file", func(t *testing.T) {
		res := batch.Result{
			Input:  buildLinkInput(),
			Status: batch.Skipped,
			File:   "out/a.txt",
		}
		line, err := JSONLine(res)
		if err != nil {
			t.Fatal(err)
		}

		want := `{"file":"out/a.txt","input":"https://youtu.be/abc","skipped":"exists"}` + "\n"
		if string(line) != want {
			t.Fatalf("line = %s, want %s", line, want)
		}
	})

	t.Run("no minutes says out_of_minutes", func(t *testing.T) {
		line, err := JSONLine(batch.Result{Input: buildLinkInput(), Status: batch.SkippedMinutes})
		if err != nil {
			t.Fatal(err)
		}

		want := `{"input":"https://youtu.be/abc","skipped":"out_of_minutes"}` + "\n"
		if string(line) != want {
			t.Fatalf("line = %s, want %s", line, want)
		}
	})
}

func TestJSONLineJobStates(t *testing.T) {
	t.Run("a failed job keeps its api error", func(t *testing.T) {
		raw := `{"id":"tr_2","status":"failed","error":{"code":"no_speech","message":"No speech"}}`
		res := batch.Result{
			Input:  buildLinkInput(),
			Status: batch.Failed,
			Err:    &api.Error{Code: api.CodeNoSpeech, Message: "No speech"},
			T:      &api.Transcription{ID: "tr_2", Raw: json.RawMessage(raw)},
		}
		line, err := JSONLine(res)
		if err != nil {
			t.Fatal(err)
		}

		if obj := decode(t, line); string(obj["error"]) != `{"code":"no_speech","message":"No speech"}` {
			t.Fatalf("line = %s", line)
		}
	})

	t.Run("an error after the job exists is added to the object", func(t *testing.T) {
		res := batch.Result{
			Input:  buildLinkInput(),
			Status: batch.Failed,
			Err:    errors.New("render failed"),
			T:      &api.Transcription{ID: "tr_3", Raw: json.RawMessage(`{"id":"tr_3","status":"completed","error":null}`)},
		}
		line, err := JSONLine(res)
		if err != nil {
			t.Fatal(err)
		}

		obj := decode(t, line)
		if string(obj["error"]) != `{"code":"error","message":"render failed"}` {
			t.Fatalf("line = %s", line)
		}

		if string(obj["id"]) != `"tr_3"` {
			t.Fatalf("line = %s", line)
		}
	})

	t.Run("a started job prints its id and status", func(t *testing.T) {
		res := batch.Result{
			Input:  buildLinkInput(),
			Status: batch.Started,
			T:      &api.Transcription{ID: "tr_4", Raw: json.RawMessage(`{"id":"tr_4","status":"queued"}`)},
		}
		line, err := JSONLine(res)
		if err != nil {
			t.Fatal(err)
		}

		obj := decode(t, line)
		if string(obj["status"]) != `"queued"` {
			t.Fatalf("line = %s", line)
		}

		if string(obj["input"]) != `"https://youtu.be/abc"` {
			t.Fatalf("line = %s", line)
		}
	})

	t.Run("an interrupted upload with an id keeps the id and gets an error", func(t *testing.T) {
		res := batch.Result{
			Input:  buildLinkInput(),
			Status: batch.Interrupted,
			T:      &api.Transcription{ID: "tr_5", Status: api.StatusAwaitingUpload},
		}
		line, err := JSONLine(res)
		if err != nil {
			t.Fatal(err)
		}

		obj := decode(t, line)
		if string(obj["id"]) != `"tr_5"` {
			t.Fatalf("line = %s", line)
		}

		if !strings.Contains(string(obj["error"]), `"code":"interrupted"`) {
			t.Fatalf("line = %s", line)
		}
	})

	t.Run("an interrupted upload is reported as an error", func(t *testing.T) {
		line, err := JSONLine(batch.Result{Input: buildLinkInput(), Status: batch.Interrupted})
		if err != nil {
			t.Fatal(err)
		}

		if obj := decode(t, line); !strings.Contains(string(obj["error"]), `"code":"interrupted"`) {
			t.Fatalf("line = %s", line)
		}
	})
}
