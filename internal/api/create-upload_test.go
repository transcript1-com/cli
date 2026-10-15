package api

import (
	"context"
	"io"
	"net/http"
	"testing"
)

const awaitingBody = `{"id":"tr_2","status":"awaiting_upload","source":{"platform":"upload","filename":"a.m4a","size":20},` +
	`"upload":{"chunk_url_template":"https://api.transcript1.com/v1/transcriptions/tr_2/upload/chunks/{index}",` +
	`"headers":{"X-Upload-Token":"tok"},"expires_at":"2026-10-15T12:00:00Z",` +
	`"chunks":[{"index":0,"offset":0,"size":10},{"index":1,"offset":10,"size":10}]}}`

func TestCreateUpload(t *testing.T) {
	var got *http.Request
	var body string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		got = r
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)
		writeJSON(w, 201, awaitingBody)
	})

	opt := CreateOptions{
		Source:   "auto",
		Language: "fr",
		Diarize:  true,
	}
	tr, err := c.CreateUpload(context.Background(), "idem-u", "a.m4a", 20, "audio/mp4", opt)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("posts the upload description", func(t *testing.T) {
		if got.Method != "POST" {
			t.Fatalf("got %s %s", got.Method, got.URL)
		}

		if got.URL.Path != "/v1/transcriptions" {
			t.Fatalf("got %s %s", got.Method, got.URL)
		}

		if got.URL.RawQuery != "" {
			t.Fatalf("got %s %s", got.Method, got.URL)
		}

		if got.Header.Get("Idempotency-Key") != "idem-u" {
			t.Fatalf("headers %v", got.Header)
		}

		want := `{"upload":{"filename":"a.m4a","size":20,"mime_type":"audio/mp4"},"language":"fr","diarize":true}`
		if body != want {
			t.Fatalf("body %s", body)
		}
	})

	t.Run("parses the chunk plan", func(t *testing.T) {
		u := tr.Upload
		if tr.Status != "awaiting_upload" {
			t.Fatalf("got %+v", tr)
		}

		if u == nil {
			t.Fatalf("got %+v", tr)
		}

		if len(u.Chunks) != 2 {
			t.Fatalf("got %+v", tr)
		}

		if u.Chunks[1].Offset != 10 {
			t.Fatalf("got %+v", tr)
		}

		if u.Headers["X-Upload-Token"] != "tok" {
			t.Fatalf("got %+v", u)
		}

		if u.ExpiresAt != "2026-10-15T12:00:00Z" {
			t.Fatalf("got %+v", u)
		}

		if tr.Source.Filename != "a.m4a" {
			t.Fatalf("got %+v", tr.Source)
		}

		if tr.Source.Size != 20 {
			t.Fatalf("got %+v", tr.Source)
		}
	})
}

func TestCreateUploadDefaultsLanguage(t *testing.T) {
	var body string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)
		writeJSON(w, 201, awaitingBody)
	})

	if _, err := c.CreateUpload(context.Background(), "i", "a.m4a", 20, "audio/mp4", CreateOptions{}); err != nil {
		t.Fatal(err)
	}

	want := `{"upload":{"filename":"a.m4a","size":20,"mime_type":"audio/mp4"},"language":"auto","diarize":false}`
	if body != want {
		t.Fatalf("body %s", body)
	}
}
