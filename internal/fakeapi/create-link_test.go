package fakeapi

import "testing"

func postLink(t *testing.T, s *Server, body string) reply {
	t.Helper()

	return call(t, "POST", s.URL+"/v1/transcriptions", s.Key, nil, []byte(body))
}

func getSource(r reply) map[string]any {
	source, _ := r.body["source"].(map[string]any)
	return source
}

func TestCreateLinkPlatforms(t *testing.T) {
	s := New(t)

	t.Run("accepts a YouTube link without a platform", func(t *testing.T) {
		r := postLink(t, s, `{"url":"https://youtu.be/dQw4w9WgXcQ"}`)
		if r.status != 202 {
			t.Fatalf("got %d %v", r.status, r.body)
		}

		if getSource(r)["platform"] != "youtube" {
			t.Fatalf("got %v", r.body)
		}
	})

	t.Run("detects an X link and returns its media id", func(t *testing.T) {
		r := postLink(t, s, `{"url":"https://x.com/jack/status/20"}`)
		if r.status != 202 {
			t.Fatalf("got %d %v", r.status, r.body)
		}

		source := getSource(r)
		if source["platform"] != "x" {
			t.Fatalf("got %v", r.body)
		}

		if source["media_id"] != "20" {
			t.Fatalf("got %v", r.body)
		}

		if source["url"] != "https://x.com/jack/status/20" {
			t.Fatalf("got %v", r.body)
		}
	})

	t.Run("a short link has a null media id", func(t *testing.T) {
		r := postLink(t, s, `{"url":"https://vm.tiktok.com/ZMabcdEF"}`)
		if r.status != 202 {
			t.Fatalf("got %d %v", r.status, r.body)
		}

		source := getSource(r)
		if source["platform"] != "tiktok" {
			t.Fatalf("got %v", r.body)
		}

		if source["media_id"] != nil {
			t.Fatalf("got %v", r.body)
		}
	})

	t.Run("a platform that is not the detected one answers invalid_url", func(t *testing.T) {
		r := postLink(t, s, `{"platform":"youtube","url":"https://vm.tiktok.com/ZMabcdEF"}`)
		e := getError(r)
		if r.status != 422 {
			t.Fatalf("got %d %v", r.status, r.body)
		}

		if e["code"] != "invalid_url" {
			t.Fatalf("got %v", r.body)
		}

		if e["message"] != "This link is from TikTok, not YouTube." {
			t.Fatalf("got %v", r.body)
		}
	})

	t.Run("an unknown platform answers unsupported_platform", func(t *testing.T) {
		r := postLink(t, s, `{"platform":"vimeo","url":"https://x.com/jack/status/20"}`)
		if r.status != 422 {
			t.Fatalf("got %d %v", r.status, r.body)
		}

		if getError(r)["code"] != "unsupported_platform" {
			t.Fatalf("got %v", r.body)
		}
	})

	t.Run("a link from another site answers invalid_url", func(t *testing.T) {
		r := postLink(t, s, `{"url":"https://vimeo.com/76979871"}`)
		e := getError(r)
		if r.status != 422 {
			t.Fatalf("got %d %v", r.status, r.body)
		}

		want := "Not a supported link. Use a YouTube, X, TikTok, Instagram, Facebook or Bluesky link."
		if e["code"] != "invalid_url" {
			t.Fatalf("got %v", r.body)
		}

		if e["message"] != want {
			t.Fatalf("got %v", r.body)
		}
	})

	t.Run("captions are refused off YouTube", func(t *testing.T) {
		r := postLink(t, s, `{"url":"https://x.com/jack/status/20","source":"captions"}`)
		if r.status != 422 {
			t.Fatalf("got %d %v", r.status, r.body)
		}

		if getError(r)["code"] != "invalid_request" {
			t.Fatalf("got %v", r.body)
		}
	})
}
