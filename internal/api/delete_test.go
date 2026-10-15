package api

import (
	"context"
	"net/http"
	"testing"
)

func TestDelete(t *testing.T) {
	t.Run("sends DELETE and accepts 204", func(t *testing.T) {
		var got *http.Request
		c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			got = r
			w.WriteHeader(http.StatusNoContent)
		})

		if err := c.Delete(context.Background(), "tr_1"); err != nil {
			t.Fatal(err)
		}

		if got.Method != "DELETE" {
			t.Fatalf("got %s %s", got.Method, got.URL.Path)
		}

		if got.URL.Path != "/v1/transcriptions/tr_1" {
			t.Fatalf("got %s %s", got.Method, got.URL.Path)
		}
	})

	t.Run("returns the API error", func(t *testing.T) {
		c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, 404, `{"error":{"code":"not_found","message":"No such transcription","retryable":false}}`)
		})

		if err := c.Delete(context.Background(), "tr_9"); !IsCode(err, CodeNotFound) {
			t.Fatalf("got %v", err)
		}
	})
}
