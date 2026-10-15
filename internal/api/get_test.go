package api

import (
	"context"
	"net/http"
	"testing"
)

func TestGet(t *testing.T) {
	var got *http.Request
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		got = r
		writeJSON(w, 200, `{"id":"tr_1","status":"failed","error":{"code":"media_private","message":"Video is private"}}`)
	})

	t.Run("sends wait when positive", func(t *testing.T) {
		tr, err := c.Get(context.Background(), "tr_1", 60)
		if err != nil {
			t.Fatal(err)
		}

		if got.Method != "GET" {
			t.Fatalf("got %s %s", got.Method, got.URL)
		}

		if got.URL.Path != "/v1/transcriptions/tr_1" {
			t.Fatalf("got %s %s", got.Method, got.URL)
		}

		if got.URL.Query().Get("wait") != "60" {
			t.Fatalf("got %s %s", got.Method, got.URL)
		}

		if tr.Error == nil {
			t.Fatalf("got %+v", tr)
		}

		if tr.Error.Code != CodeMediaPrivate {
			t.Fatalf("got %+v", tr)
		}

		if !tr.Terminal() {
			t.Fatalf("got %+v", tr)
		}
	})

	t.Run("omits wait when zero", func(t *testing.T) {
		if _, err := c.Get(context.Background(), "tr_1", 0); err != nil {
			t.Fatal(err)
		}

		if got.URL.RawQuery != "" {
			t.Fatalf("query %q", got.URL.RawQuery)
		}
	})

	t.Run("escapes the id in the path", func(t *testing.T) {
		if _, err := c.Get(context.Background(), "tr_1/../x", 0); err != nil {
			t.Fatal(err)
		}

		if got.URL.EscapedPath() != "/v1/transcriptions/tr_1%2F..%2Fx" {
			t.Fatalf("path %q", got.URL.EscapedPath())
		}
	})
}
