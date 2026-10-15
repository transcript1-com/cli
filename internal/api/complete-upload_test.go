package api

import (
	"context"
	"net/http"
	"slices"
	"testing"
)

func TestCompleteUpload(t *testing.T) {
	t.Run("posts to the complete endpoint and returns the queued job", func(t *testing.T) {
		var got *http.Request
		c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			got = r
			writeJSON(w, 202, `{"id":"tr_2","status":"queued"}`)
		})

		tr, err := c.CompleteUpload(context.Background(), "tr_2")
		if err != nil {
			t.Fatal(err)
		}

		if got.Method != "POST" {
			t.Fatalf("got %s %s, %+v", got.Method, got.URL.Path, tr)
		}

		if got.URL.Path != "/v1/transcriptions/tr_2/upload/complete" {
			t.Fatalf("got %s %s, %+v", got.Method, got.URL.Path, tr)
		}

		if tr.Status != "queued" {
			t.Fatalf("got %s %s, %+v", got.Method, got.URL.Path, tr)
		}
	})

	t.Run("reports missing chunks in the error details", func(t *testing.T) {
		c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, 409, `{"error":{"code":"upload_incomplete","message":"Chunks missing.","retryable":false,"missing_chunks":[1,3]}}`)
		})

		_, err := c.CompleteUpload(context.Background(), "tr_2")
		apiErr := getAPIError(t, err)
		if !IsCode(err, CodeUploadIncomplete) {
			t.Fatalf("got %v", err)
		}

		missing, ok := apiErr.Details["missing_chunks"].([]any)
		if !ok {
			t.Fatalf("got %v", apiErr.Details)
		}

		if !slices.Equal(missing, []any{1.0, 3.0}) {
			t.Fatalf("got %v", apiErr.Details)
		}
	})
}
