package inputs

import (
	"encoding/json"
	"os"
	"testing"
)

const (
	wantID        = "dQw4w9WgXcQ"
	wantCanonical = "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
)

func TestYouTubeVectors(t *testing.T) {
	raw, err := os.ReadFile("../../testdata/youtube-urls.json")
	if err != nil {
		t.Fatal(err)
	}

	var vectors struct {
		Accept []string `json:"accept"`
		Reject []string `json:"reject"`
	}
	if err := json.Unmarshal(raw, &vectors); err != nil {
		t.Fatal(err)
	}

	if len(vectors.Accept) != 9 {
		t.Fatalf("got %d accept vectors, want 9", len(vectors.Accept))
	}

	if len(vectors.Reject) != 9 {
		t.Fatalf("got %d reject vectors, want 9", len(vectors.Reject))
	}

	for _, u := range vectors.Accept {
		t.Run("accepts "+u, func(t *testing.T) {
			id, canonical, ok := ParseYouTube(u)
			if !ok {
				t.Fatalf("ParseYouTube(%q) = %q, %q, false", u, id, canonical)
			}

			if id != wantID {
				t.Fatalf("ParseYouTube(%q) id = %q, want %q", u, id, wantID)
			}

			if canonical != wantCanonical {
				t.Fatalf("ParseYouTube(%q) canonical = %q, want %q", u, canonical, wantCanonical)
			}
		})
	}

	for _, u := range vectors.Reject {
		t.Run("rejects "+u, func(t *testing.T) {
			if id, canonical, ok := ParseYouTube(u); ok {
				t.Fatalf("ParseYouTube(%q) = %q, %q, true; want false", u, id, canonical)
			}
		})
	}
}
