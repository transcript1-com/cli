package fakeapi

import "testing"

func TestCompleteAtTheActiveLimit(t *testing.T) {
	s := New(t)
	s.ActiveLimit = 1
	link := []byte(`{"platform":"youtube","url":"https://youtu.be/dQw4w9WgXcQ"}`)
	held := call(t, "POST", s.URL+"/v1/transcriptions", s.Key, nil, link)
	created := createUpload(t, s, "k1")
	id, _ := created.body["id"].(string)
	for index, data := range []string{"01234567", "89abcdef", "ghij"} {
		url := s.URL + "/v1/transcriptions/" + id + "/upload/chunks/" + string(rune('0'+index))
		call(t, "PUT", url, "", map[string]string{"X-Upload-Token": getUploadToken(created)}, []byte(data))
	}

	complete := s.URL + "/v1/transcriptions/" + id + "/upload/complete"
	blocked := call(t, "POST", complete, s.Key, nil, nil)

	t.Run("answers 429 too_many_active_jobs with Retry-After", func(t *testing.T) {
		if blocked.status != 429 {
			t.Fatalf("got %d %v", blocked.status, blocked.body)
		}

		if getError(blocked)["code"] != "too_many_active_jobs" {
			t.Fatalf("got %d %v", blocked.status, blocked.body)
		}

		if blocked.header.Get("Retry-After") != "1" {
			t.Fatalf("retry after %q", blocked.header.Get("Retry-After"))
		}
	})

	t.Run("keeps the upload awaiting so complete can be retried", func(t *testing.T) {
		call(t, "GET", s.URL+"/v1/transcriptions/"+held.body["id"].(string), s.Key, nil, nil)
		again := call(t, "POST", complete, s.Key, nil, nil)
		if again.status != 202 {
			t.Fatalf("got %d %v", again.status, again.body)
		}

		if again.body["status"] != "queued" {
			t.Fatalf("got %d %v", again.status, again.body)
		}
	})
}
