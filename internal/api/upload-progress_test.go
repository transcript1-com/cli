package api

import (
	"context"
	"net/http"
	"slices"
	"testing"
)

func TestUploadProgress(t *testing.T) {
	var got *http.Request
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		got = r
		writeJSON(w, 200, `{"object":"upload","status":"awaiting_upload","total_bytes":20,"uploaded_bytes":10,`+
			`"received_chunks":[0],"chunks":[{"index":0,"offset":0,"size":10},{"index":1,"offset":10,"size":10}],"expires_at":null}`)
	})

	p, err := c.UploadProgress(context.Background(), "tr_2")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("reads the upload resource", func(t *testing.T) {
		if got.Method != "GET" {
			t.Fatalf("got %s %s", got.Method, got.URL.Path)
		}

		if got.URL.Path != "/v1/transcriptions/tr_2/upload" {
			t.Fatalf("got %s %s", got.Method, got.URL.Path)
		}
	})

	t.Run("parses received chunks and bytes", func(t *testing.T) {
		if p.UploadedBytes != 10 {
			t.Fatalf("got %+v", p)
		}

		if !slices.Equal(p.ReceivedChunks, []int{0}) {
			t.Fatalf("got %+v", p)
		}

		if len(p.Chunks) != 2 {
			t.Fatalf("got %+v", p)
		}
	})
}
