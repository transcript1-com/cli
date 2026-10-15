package api

import (
	"context"
	"net/http"
	"testing"
)

func TestRender(t *testing.T) {
	var got *http.Request
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		got = r
		w.Header().Set("Content-Type", "application/x-subrip")
		_, _ = w.Write([]byte("1\n00:00:00,000 --> 00:00:03,200\nHello\n"))
	})

	t.Run("returns the rendered body bytes", func(t *testing.T) {
		out, err := c.Render(context.Background(), "tr_1", "srt", false)
		if err != nil {
			t.Fatal(err)
		}

		if got.URL.Path != "/v1/transcriptions/tr_1" {
			t.Fatalf("got %s", got.URL)
		}

		if got.URL.Query().Get("format") != "srt" {
			t.Fatalf("got %s", got.URL)
		}

		if got.URL.Query().Has("timestamps") {
			t.Fatalf("query %q, body %q", got.URL.RawQuery, out)
		}

		if string(out) != "1\n00:00:00,000 --> 00:00:03,200\nHello\n" {
			t.Fatalf("query %q, body %q", got.URL.RawQuery, out)
		}
	})

	t.Run("asks for timestamps on text", func(t *testing.T) {
		if _, err := c.Render(context.Background(), "tr_1", "text", true); err != nil {
			t.Fatal(err)
		}

		q := got.URL.Query()
		if q.Get("format") != "text" {
			t.Fatalf("query %v", q)
		}

		if q.Get("timestamps") != "true" {
			t.Fatalf("query %v", q)
		}
	})

	t.Run("leaves timestamps out for other formats", func(t *testing.T) {
		if _, err := c.Render(context.Background(), "tr_1", "vtt", true); err != nil {
			t.Fatal(err)
		}

		if got.URL.Query().Has("timestamps") {
			t.Fatalf("query %v", got.URL.Query())
		}
	})
}
