package api

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

func TestPutChunkHeaders(t *testing.T) {
	var got *http.Request
	var received []byte
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		got = r
		received, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNoContent)
	})

	section := io.NewSectionReader(strings.NewReader("0123456789abcdef"), 4, 8)

	if err := c.PutChunk(context.Background(), c.BaseURL+"/v1/transcriptions/tr_1/upload/chunks/1", "tok", section); err != nil {
		t.Fatal(err)
	}

	t.Run("sends a PUT with the token and no bearer", func(t *testing.T) {
		if got.Method != "PUT" {
			t.Fatalf("got %s %s", got.Method, got.URL.Path)
		}

		if got.URL.Path != "/v1/transcriptions/tr_1/upload/chunks/1" {
			t.Fatalf("got %s %s", got.Method, got.URL.Path)
		}

		if got.Header.Get("X-Upload-Token") != "tok" {
			t.Fatalf("headers %v", got.Header)
		}

		if got.Header.Get("Authorization") != "" {
			t.Fatalf("headers %v", got.Header)
		}

		if got.Header.Get("Content-Type") != "application/octet-stream" {
			t.Fatalf("content type %q", got.Header.Get("Content-Type"))
		}

		if got.Header.Get("User-Agent") != "transcript1-cli/test (linux/amd64)" {
			t.Fatalf("user agent %q", got.Header.Get("User-Agent"))
		}
	})

	t.Run("declares the exact length and sends only the section", func(t *testing.T) {
		if got.ContentLength != 8 {
			t.Fatalf("length %d, body %q", got.ContentLength, received)
		}

		if string(received) != "456789ab" {
			t.Fatalf("length %d, body %q", got.ContentLength, received)
		}
	})
}

func TestPutChunkRetries(t *testing.T) {
	var calls atomic.Int32
	var bodies []string
	c, s := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(raw))
		if calls.Add(1) == 1 {
			writeJSON(w, 429, `{"error":{"code":"rate_limited","message":"Slow down","retryable":true}}`)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})

	section := io.NewSectionReader(bytes.NewReader([]byte("chunk-data")), 0, 10)

	t.Run("sends the whole body again on every attempt", func(t *testing.T) {
		if err := c.PutChunk(context.Background(), c.BaseURL+"/c/0", "tok", section); err != nil {
			t.Fatal(err)
		}

		if len(bodies) != 2 {
			t.Fatalf("bodies %q, sleeps %v", bodies, s.slept)
		}

		if bodies[0] != "chunk-data" {
			t.Fatalf("bodies %q, sleeps %v", bodies, s.slept)
		}

		if bodies[1] != "chunk-data" {
			t.Fatalf("bodies %q, sleeps %v", bodies, s.slept)
		}

		if len(s.slept) != 1 {
			t.Fatalf("bodies %q, sleeps %v", bodies, s.slept)
		}
	})

	t.Run("reports an expired token as unauthorized", func(t *testing.T) {
		c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, 401, `{"error":{"code":"unauthorized","message":"Upload token expired","retryable":false}}`)
		})

		err := c.PutChunk(context.Background(), c.BaseURL+"/c/0", "old", section)
		if !IsCode(err, CodeUnauthorized) {
			t.Fatalf("got %v", err)
		}
	})
}
