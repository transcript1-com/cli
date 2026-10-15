package api

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"
)

const reusedBody = `{"id":"tr_1","object":"transcription","status":"completed","reused":true,` +
	`"source":{"platform":"youtube","url":"https://www.youtube.com/watch?v=dQw4w9WgXcQ","media_id":"dQw4w9WgXcQ"},` +
	`"media":{"title":"Song","duration_seconds":212.5},"transcript_source":"youtube_captions","language":"en",` +
	`"billed_minutes":4,"error":null}`

func TestCreateLinkRequest(t *testing.T) {
	var got *http.Request
	var body string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		got = r
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)
		writeJSON(w, 202, `{"id":"tr_1","status":"queued"}`)
	})

	url := "https://www.youtube.com/watch?v=dQw4w9WgXcQ"

	t.Run("posts the link with wait and reuse", func(t *testing.T) {
		if _, err := c.CreateLink(context.Background(), "idem-1", url, CreateOptions{}, 60, true); err != nil {
			t.Fatal(err)
		}

		if got.Method != "POST" {
			t.Fatalf("got %s %s", got.Method, got.URL.Path)
		}

		if got.URL.Path != "/v1/transcriptions" {
			t.Fatalf("got %s %s", got.Method, got.URL.Path)
		}

		q := got.URL.Query()
		if q.Get("wait") != "60" {
			t.Fatalf("query %v", q)
		}

		if q.Get("reuse_recent") != "true" {
			t.Fatalf("query %v", q)
		}

		if got.Header.Get("Authorization") != "Bearer t1_x" {
			t.Fatalf("headers %v", got.Header)
		}

		if got.Header.Get("Idempotency-Key") != "idem-1" {
			t.Fatalf("headers %v", got.Header)
		}

		if got.Header.Get("User-Agent") != "transcript1-cli/test (linux/amd64)" {
			t.Fatalf("user agent %q", got.Header.Get("User-Agent"))
		}

		want := `{"platform":"youtube","url":"` + url + `","source":"auto","language":"auto","diarize":false}`
		if body != want {
			t.Fatalf("body %s", body)
		}
	})

	t.Run("omits reuse_recent when not asked for and sends wait=0", func(t *testing.T) {
		if _, err := c.CreateLink(context.Background(), "idem-2", url, CreateOptions{}, 0, false); err != nil {
			t.Fatal(err)
		}

		q := got.URL.Query()
		if q.Has("reuse_recent") {
			t.Fatalf("query %v", q)
		}

		if q.Get("wait") != "0" {
			t.Fatalf("query %v", q)
		}
	})

	t.Run("sends the chosen options", func(t *testing.T) {
		opt := CreateOptions{
			Source:   "stt",
			Language: "de",
			Diarize:  true,
		}
		if _, err := c.CreateLink(context.Background(), "idem-3", url, opt, 0, false); err != nil {
			t.Fatal(err)
		}

		want := `{"platform":"youtube","url":"` + url + `","source":"stt","language":"de","diarize":true}`
		if body != want {
			t.Fatalf("body %s", body)
		}
	})
}

func TestTranscriptionRawKept(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, reusedBody)
	})

	tr, err := c.CreateLink(context.Background(), "idem", "https://youtu.be/dQw4w9WgXcQ", CreateOptions{}, 60, true)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("keeps the exact response body", func(t *testing.T) {
		if string(tr.Raw) != reusedBody {
			t.Fatalf("raw %s", tr.Raw)
		}
	})

	t.Run("parses the fields the cli reads", func(t *testing.T) {
		if !tr.Reused {
			t.Fatalf("got %+v", tr)
		}

		if tr.ID != "tr_1" {
			t.Fatalf("got %+v", tr)
		}

		if !tr.Terminal() {
			t.Fatalf("got %+v", tr)
		}

		if tr.Source.MediaID != "dQw4w9WgXcQ" {
			t.Fatalf("got %+v", tr)
		}

		if tr.Media.Title != "Song" {
			t.Fatalf("got %+v", tr)
		}

		if *tr.Media.DurationSeconds != 212.5 {
			t.Fatalf("got %+v", tr)
		}

		if *tr.BilledMinutes != 4 {
			t.Fatalf("got %+v", tr)
		}

		if *tr.Language != "en" {
			t.Fatalf("got %+v", tr)
		}

		if *tr.TranscriptSource != "youtube_captions" {
			t.Fatalf("got %+v", tr)
		}
	})
}

func TestCreateDoesNotRetry429(t *testing.T) {
	var calls int
	c, s := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Retry-After", "30")
		writeJSON(w, 429, `{"error":{"code":"too_many_active_jobs","message":"Too many jobs","retryable":true}}`)
	})

	_, err := c.CreateLink(context.Background(), "idem", "https://youtu.be/dQw4w9WgXcQ", CreateOptions{}, 0, false)
	apiErr := getAPIError(t, err)
	if calls != 1 {
		t.Fatalf("calls %d, sleeps %v", calls, s.slept)
	}

	if len(s.slept) != 0 {
		t.Fatalf("calls %d, sleeps %v", calls, s.slept)
	}

	if apiErr.Code != CodeTooManyActiveJobs {
		t.Fatalf("got %+v", apiErr)
	}

	if apiErr.RetryAfter != 30*time.Second {
		t.Fatalf("got %+v", apiErr)
	}

	if !apiErr.Retryable {
		t.Fatalf("got %+v", apiErr)
	}
}

func TestTerminal(t *testing.T) {
	for status, want := range map[string]bool{
		StatusCompleted:   true,
		StatusFailed:      true,
		StatusCanceled:    true,
		"queued":          false,
		"processing":      false,
		"awaiting_upload": false,
	} {
		t.Run(status, func(t *testing.T) {
			if (&Transcription{Status: status}).Terminal() != want {
				t.Fatalf("Terminal() != %v", want)
			}
		})
	}
}
