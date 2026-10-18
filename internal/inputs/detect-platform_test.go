package inputs

import (
	"encoding/json"
	"os"
	"testing"
)

const errUnsupported = ": Only YouTube, X, TikTok, Instagram, Facebook and Bluesky links and files are supported"

type socialVectorsType struct {
	Accept []struct {
		URL      string `json:"url"`
		Platform string `json:"platform"`
	} `json:"accept"`
	Reject  []string `json:"reject"`
	Foreign []string `json:"foreign"`
}

func loadSocialVectors(t *testing.T) socialVectorsType {
	t.Helper()
	raw, err := os.ReadFile("../../testdata/social-urls.json")
	if err != nil {
		t.Fatal(err)
	}

	var vectors socialVectorsType
	if err := json.Unmarshal(raw, &vectors); err != nil {
		t.Fatal(err)
	}

	return vectors
}

func TestSocialVectors(t *testing.T) {
	vectors := loadSocialVectors(t)
	if len(vectors.Accept) == 0 {
		t.Fatal("no accept vectors")
	}

	if len(vectors.Reject) == 0 {
		t.Fatal("no reject vectors")
	}

	if len(vectors.Foreign) == 0 {
		t.Fatal("no foreign vectors")
	}

	for _, v := range vectors.Accept {
		t.Run("accepts "+v.URL, func(t *testing.T) {
			assertSocialLink(t, v.URL, v.Platform)
		})
	}

	// the host check is no stricter than the api, which rejects bad paths itself
	for _, u := range vectors.Reject {
		t.Run("passes on "+u[:min(len(u), 60)], func(t *testing.T) {
			platform, ok := DetectPlatform(u)
			if !ok {
				t.Fatalf("DetectPlatform(%q) = %q, false", u, platform)
			}

			assertSocialLink(t, u, platform)
		})
	}

	for _, u := range vectors.Foreign {
		t.Run("refuses "+u, func(t *testing.T) {
			if platform, ok := DetectPlatform(u); ok {
				t.Fatalf("DetectPlatform(%q) = %q, true; want false", u, platform)
			}

			_, err := newLink(u)
			if err == nil {
				t.Fatalf("newLink(%q) succeeded", u)
			}

			if err.Error() != u+errUnsupported {
				t.Fatalf("got %q", err.Error())
			}
		})
	}
}

func assertSocialLink(t *testing.T, u, platform string) {
	t.Helper()
	in, err := newLink(u)
	if err != nil {
		t.Fatal(err)
	}

	if in.Platform != platform {
		t.Fatalf("platform = %q, want %q", in.Platform, platform)
	}

	if in.Kind != Link {
		t.Fatalf("kind = %v", in.Kind)
	}

	if in.Raw != u {
		t.Fatalf("raw = %q", in.Raw)
	}

	if in.URL != u {
		t.Fatalf("url = %q, want the raw link", in.URL)
	}

	if in.VideoID != "" {
		t.Fatalf("video id = %q, want empty", in.VideoID)
	}
}

func TestDetectPlatformTrimsSpaces(t *testing.T) {
	in, err := newLink("  https://x.com/jack/status/20 \n")
	if err != nil {
		t.Fatal(err)
	}

	if in.URL != "https://x.com/jack/status/20" {
		t.Fatalf("url = %q", in.URL)
	}

	if in.Platform != "x" {
		t.Fatalf("platform = %q", in.Platform)
	}
}

func TestYouTubeLinkInput(t *testing.T) {
	// 1.1.0 stripped www. from every YouTube host
	links := []string{
		"https://youtu.be/dQw4w9WgXcQ?si=x",
		"https://www.youtu.be/dQw4w9WgXcQ",
		"https://www.m.youtube.com/watch?v=dQw4w9WgXcQ",
		"https://www.music.youtube.com/watch?v=dQw4w9WgXcQ",
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ",
	}
	for _, u := range links {
		t.Run(u, func(t *testing.T) {
			if platform, ok := DetectPlatform(u); !ok || platform != "youtube" {
				t.Fatalf("DetectPlatform = %q, %v", platform, ok)
			}

			in, err := newLink(u)
			if err != nil {
				t.Fatal(err)
			}

			want := Input{
				Raw:      u,
				Kind:     Link,
				Platform: "youtube",
				URL:      wantCanonical,
				VideoID:  wantID,
			}
			if in != want {
				t.Fatalf("got %+v", in)
			}
		})
	}
}

func TestYouTubeHostWithoutVideoIsRefused(t *testing.T) {
	u := "https://www.youtube.com/@RickAstleyYT"
	_, err := newLink(u)
	if err == nil {
		t.Fatal("newLink succeeded")
	}

	if err.Error() != u+errUnsupported {
		t.Fatalf("got %q", err.Error())
	}
}

func TestShortLinkHostsAreAccepted(t *testing.T) {
	cases := map[string]string{
		"https://vm.tiktok.com/ZMabcdEF":              "tiktok",
		"https://vt.tiktok.com/ZSabc123":              "tiktok",
		"https://www.tiktok.com/t/ZTabcd12":           "tiktok",
		"https://fb.watch/AbC_12-x/":                  "facebook",
		"https://www.facebook.com/share/v/1AbCdEfGh/": "facebook",
		"https://m.facebook.com/share/r/1AbCdEfGh/":   "facebook",
	}
	for u, want := range cases {
		t.Run(u, func(t *testing.T) {
			assertSocialLink(t, u, want)
		})
	}
}
