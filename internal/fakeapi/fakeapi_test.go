package fakeapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

type reply struct {
	status int
	header http.Header
	body   map[string]any
}

func call(t *testing.T, method, url, key string, header map[string]string, body []byte) reply {
	t.Helper()
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}

	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}

	for name, value := range header {
		req.Header.Set(name, value)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	out := reply{status: res.StatusCode, header: res.Header}
	_ = json.Unmarshal(raw, &out.body)

	return out
}

func getError(r reply) map[string]any {
	e, _ := r.body["error"].(map[string]any)
	return e
}

func createUpload(t *testing.T, s *Server, key string) reply {
	t.Helper()
	body := []byte(`{"upload":{"filename":"a.mp3","size":20,"mime_type":"audio/mpeg"},"language":"auto","diarize":false}`)

	return call(t, "POST", s.URL+"/v1/transcriptions", s.Key, map[string]string{"Idempotency-Key": key}, body)
}

func getUploadToken(r reply) string {
	upload, _ := r.body["upload"].(map[string]any)
	headers, _ := upload["headers"].(map[string]any)
	token, _ := headers["X-Upload-Token"].(string)

	return token
}

func TestEnvelope(t *testing.T) {
	s := New(t)

	t.Run("a wrong key answers 401 unauthorized in the error envelope", func(t *testing.T) {
		r := call(t, "GET", s.URL+"/v1/account", "t1_wrong", nil, nil)
		e := getError(r)
		if r.status != 401 {
			t.Fatalf("got %d %v", r.status, r.body)
		}

		if e["code"] != "unauthorized" {
			t.Fatalf("got %d %v", r.status, r.body)
		}

		if e["retryable"] != false {
			t.Fatalf("got %d %v", r.status, r.body)
		}

		if e["message"] == "" {
			t.Fatalf("got %d %v", r.status, r.body)
		}

		if r.header.Get("WWW-Authenticate") == "" {
			t.Fatal("no WWW-Authenticate header")
		}
	})

	t.Run("an unknown id answers 404 not_found", func(t *testing.T) {
		r := call(t, "GET", s.URL+"/v1/transcriptions/tr_nope", s.Key, nil, nil)
		if r.status != 404 {
			t.Fatalf("got %d %v", r.status, r.body)
		}

		if getError(r)["code"] != "not_found" {
			t.Fatalf("got %d %v", r.status, r.body)
		}
	})

	t.Run("the account shows the demo user and unlimited minutes as 1000", func(t *testing.T) {
		r := call(t, "GET", s.URL+"/v1/account", s.Key, nil, nil)
		user, _ := r.body["user"].(map[string]any)
		plan, _ := r.body["plan"].(map[string]any)
		minutes, _ := r.body["minutes"].(map[string]any)
		if user["email"] != "demo@transcript1.com" {
			t.Fatalf("got %v", r.body)
		}

		if plan["name"] != "Monthly" {
			t.Fatalf("got %v", r.body)
		}

		if minutes["remaining"] != 1000.0 {
			t.Fatalf("got %v", r.body)
		}
	})

	t.Run("an active job limit answers 429 too_many_active_jobs with Retry-After", func(t *testing.T) {
		s := New(t)
		s.ActiveLimit = 1
		link := []byte(`{"platform":"youtube","url":"https://youtu.be/dQw4w9WgXcQ"}`)
		first := call(t, "POST", s.URL+"/v1/transcriptions", s.Key, nil, link)
		second := call(t, "POST", s.URL+"/v1/transcriptions", s.Key, nil, link)
		if first.status != 202 {
			t.Fatalf("got %d, %d %v", first.status, second.status, second.body)
		}

		if second.status != 429 {
			t.Fatalf("got %d, %d %v", first.status, second.status, second.body)
		}

		if getError(second)["code"] != "too_many_active_jobs" {
			t.Fatalf("got %d, %d %v", first.status, second.status, second.body)
		}

		if second.header.Get("Retry-After") != "1" {
			t.Fatalf("retry after %q, peak %d", second.header.Get("Retry-After"), s.PeakActive())
		}

		if s.PeakActive() != 1 {
			t.Fatalf("retry after %q, peak %d", second.header.Get("Retry-After"), s.PeakActive())
		}
	})
}

func TestTokenRotation(t *testing.T) {
	s := New(t)
	first := createUpload(t, s, "k1")
	second := createUpload(t, s, "k1")
	id, _ := first.body["id"].(string)
	chunk := s.URL + "/v1/transcriptions/" + id + "/upload/chunks/0"

	t.Run("the same key returns the same upload with a new token", func(t *testing.T) {
		if first.status != 201 {
			t.Fatalf("got %d %v, %d %v", first.status, first.body, second.status, second.body)
		}

		if second.status != 201 {
			t.Fatalf("got %d %v, %d %v", first.status, first.body, second.status, second.body)
		}

		if second.body["id"] != id {
			t.Fatalf("got %d %v, %d %v", first.status, first.body, second.status, second.body)
		}

		if getUploadToken(first) == "" {
			t.Fatalf("tokens %q, %q", getUploadToken(first), getUploadToken(second))
		}

		if getUploadToken(first) == getUploadToken(second) {
			t.Fatalf("tokens %q, %q", getUploadToken(first), getUploadToken(second))
		}
	})

	t.Run("the old token is refused and the new one accepted", func(t *testing.T) {
		old := call(t, "PUT", chunk, "", map[string]string{"X-Upload-Token": getUploadToken(first)}, []byte("01234567"))
		if old.status != 401 {
			t.Fatalf("got %d %v", old.status, old.body)
		}

		if getError(old)["code"] != "unauthorized" {
			t.Fatalf("got %d %v", old.status, old.body)
		}

		ok := call(t, "PUT", chunk, "", map[string]string{"X-Upload-Token": getUploadToken(second)}, []byte("01234567"))
		if ok.status != 200 {
			t.Fatalf("got %d %v", ok.status, ok.body)
		}
	})

	t.Run("progress lists the received chunk", func(t *testing.T) {
		r := call(t, "GET", s.URL+"/v1/transcriptions/"+id+"/upload", s.Key, nil, nil)
		received, _ := r.body["received_chunks"].([]any)
		if r.status != 200 {
			t.Fatalf("got %d %v", r.status, r.body)
		}

		if len(received) != 1 {
			t.Fatalf("got %d %v", r.status, r.body)
		}

		if received[0] != 0.0 {
			t.Fatalf("got %d %v", r.status, r.body)
		}

		if r.body["uploaded_bytes"] != 8.0 {
			t.Fatalf("got %d %v", r.status, r.body)
		}
	})

	t.Run("other parameters with the same key answer 409 idempotency_conflict", func(t *testing.T) {
		body := []byte(`{"upload":{"filename":"b.mp3","size":20,"mime_type":"audio/mpeg"}}`)
		r := call(t, "POST", s.URL+"/v1/transcriptions", s.Key, map[string]string{"Idempotency-Key": "k1"}, body)
		if r.status != 409 {
			t.Fatalf("got %d %v", r.status, r.body)
		}

		if getError(r)["code"] != "idempotency_conflict" {
			t.Fatalf("got %d %v", r.status, r.body)
		}
	})
}
