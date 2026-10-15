package batch

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/transcript1/cli/internal/api"
)

const (
	wantText = "Welcome to the show.\nThanks for having me.\n"
	postPath = "POST /v1/transcriptions"
	getPath  = "GET /v1/transcriptions/tr_1"
)

func TestLinkText(t *testing.T) {
	srv, c := setup(t)
	res := run(t, c, buildLink(rickID), Options{})
	reqs := srv.Requests()

	t.Run("creates with reuse and wait, polls with wait, then renders text", func(t *testing.T) {
		if want := []string{postPath, getPath, getPath}; !slices.Equal(srv.Log(), want) {
			t.Fatalf("log %v", srv.Log())
		}

		if reqs[0].Query.Get("reuse_recent") != "true" {
			t.Fatalf("create query %v", reqs[0].Query)
		}

		if reqs[0].Query.Get("wait") != "60" {
			t.Fatalf("create query %v", reqs[0].Query)
		}

		if reqs[1].Query.Get("wait") != "60" {
			t.Fatalf("queries %v %v", reqs[1].Query, reqs[2].Query)
		}

		if reqs[2].Query.Get("format") != "text" {
			t.Fatalf("queries %v %v", reqs[1].Query, reqs[2].Query)
		}
	})

	t.Run("returns the text as the body", func(t *testing.T) {
		if res.Status != Done {
			t.Fatalf("result %+v", res)
		}

		if res.Err != nil {
			t.Fatalf("result %+v", res)
		}

		if res.T.Status != api.StatusCompleted {
			t.Fatalf("result %+v", res)
		}

		if string(res.Body) != wantText {
			t.Fatalf("body %q", res.Body)
		}
	})
}

func TestLinkFresh(t *testing.T) {
	srv, c := setup(t)
	run(t, c, buildLink(rickID), Options{Fresh: true})
	if srv.Requests()[0].Query.Has("reuse_recent") {
		t.Fatalf("query %v", srv.Requests()[0].Query)
	}
}

func TestLinkJSONFormat(t *testing.T) {
	srv, c := setup(t)
	res := run(t, c, buildLink(rickID), Options{Format: "json"})

	t.Run("asks for no rendered body", func(t *testing.T) {
		if want := []string{postPath, getPath}; !slices.Equal(srv.Log(), want) {
			t.Fatalf("log %v", srv.Log())
		}
	})

	t.Run("returns the transcription object with one trailing newline", func(t *testing.T) {
		want := strings.TrimRight(string(res.T.Raw), "\n") + "\n"
		if string(res.Body) != want {
			t.Fatalf("body %q", res.Body)
		}

		if !strings.Contains(want, `"transcript"`) {
			t.Fatalf("body %q", res.Body)
		}
	})
}

func TestReusedJSONFetchesTheTranscript(t *testing.T) {
	srv, c := setup(t)
	run(t, c, buildLink(rickID), Options{})
	res := run(t, c, buildLink(rickID), Options{Format: "json"})
	if res.Status != Done {
		t.Fatalf("result %+v body %q", res, res.Body)
	}

	if !res.T.Reused {
		t.Fatalf("result %+v body %q", res, res.Body)
	}

	if !strings.Contains(string(res.Body), `"transcript"`) {
		t.Fatalf("result %+v body %q", res, res.Body)
	}

	if len(srv.Log()) != 5 {
		t.Fatalf("log %v", srv.Log())
	}
}

func TestJSONToStdoutSkipsTheRender(t *testing.T) {
	srv, c := setup(t)
	res := run(t, c, buildLink(rickID), Options{JSON: true})
	if want := []string{postPath, getPath}; !slices.Equal(srv.Log(), want) {
		t.Fatalf("log %v", srv.Log())
	}

	if res.Status != Done {
		t.Fatalf("result %+v", res)
	}

	if len(res.Body) != 0 {
		t.Fatalf("result %+v", res)
	}

	if !strings.Contains(string(res.T.Raw), `"transcript"`) {
		t.Fatalf("result %+v", res)
	}

	for _, req := range srv.Requests() {
		if req.Query.Has("format") {
			t.Fatalf("render request %v", req.Query)
		}
	}
}

func TestReusedJSONToStdoutHasTheTranscript(t *testing.T) {
	_, c := setup(t)
	run(t, c, buildLink(rickID), Options{})
	res := run(t, c, buildLink(rickID), Options{JSON: true})
	if res.Status != Done {
		t.Fatalf("result %+v", res)
	}

	if !res.T.Reused {
		t.Fatalf("result %+v", res)
	}

	if !strings.Contains(string(res.T.Raw), `"transcript"`) {
		t.Fatalf("result %+v", res)
	}
}

func TestOptionsReachTheAPI(t *testing.T) {
	srv, c := setup(t)
	opt := Options{
		Timestamps: true,
		Speakers:   true,
		Language:   "de",
		Source:     "stt",
	}
	res := run(t, c, buildLink(rickID), opt)
	reqs := srv.Requests()

	t.Run("sends them in the create body", func(t *testing.T) {
		var body map[string]any
		if err := json.Unmarshal(reqs[0].Body, &body); err != nil {
			t.Fatal(err)
		}

		if body["diarize"] != true {
			t.Fatalf("body %v", body)
		}

		if body["language"] != "de" {
			t.Fatalf("body %v", body)
		}

		if body["source"] != "stt" {
			t.Fatalf("body %v", body)
		}
	})

	t.Run("asks for timestamps when rendering", func(t *testing.T) {
		if reqs[2].Query.Get("timestamps") != "true" {
			t.Fatalf("query %v body %q", reqs[2].Query, res.Body)
		}

		if !strings.Contains(string(res.Body), "[00:00:00] Speaker 1:") {
			t.Fatalf("query %v body %q", reqs[2].Query, res.Body)
		}
	})

	t.Run("sends no source other than auto for a file", func(t *testing.T) {
		srv, c := setup(t)
		run(t, c, buildFileInput(t, "talk.mp3"), opt)
		var body map[string]any
		if err := json.Unmarshal(srv.Requests()[0].Body, &body); err != nil {
			t.Fatal(err)
		}

		source, ok := body["source"]
		if !ok {
			return
		}

		if source != "auto" {
			t.Fatalf("body %v", body)
		}
	})
}

func TestFileUpload(t *testing.T) {
	srv, c := setup(t)
	var ev events
	res := runOne(context.Background(), c, 3, buildFileInput(t, "talk.mp3"), Options{}, false, ev.emit)

	t.Run("uploads, completes and then waits", func(t *testing.T) {
		if res.Status != Done {
			t.Fatalf("result %+v", res)
		}

		if res.Err != nil {
			t.Fatalf("result %+v", res)
		}

		log := srv.Log()
		if log[0] != postPath {
			t.Fatalf("log %v", log)
		}

		if log[len(log)-1] != "GET /v1/transcriptions/tr_1" {
			t.Fatalf("log %v", log)
		}
	})

	t.Run("emits uploading, queued and finished in order", func(t *testing.T) {
		stages := ev.listStages()
		if !ev.has(Uploading) {
			t.Fatalf("stages %v", stages)
		}

		if !ev.has(Queued) {
			t.Fatalf("stages %v", stages)
		}

		if stages[len(stages)-1] != Finished {
			t.Fatalf("stages %v", stages)
		}

		if slices.Index(stages, Uploading) > slices.Index(stages, Queued) {
			t.Fatalf("stages %v", stages)
		}
	})

	t.Run("reports the upload size and the result", func(t *testing.T) {
		for _, e := range ev.list {
			if e.Index != 3 {
				t.Fatalf("event %+v", e)
			}

			if e.Stage != Uploading {
				continue
			}

			if e.Total != 20 {
				t.Fatalf("event %+v", e)
			}
		}

		last := ev.list[len(ev.list)-1]
		if last.Result == nil {
			t.Fatalf("last %+v", last)
		}

		if last.Result.T != res.T {
			t.Fatalf("last %+v", last)
		}
	})
}
