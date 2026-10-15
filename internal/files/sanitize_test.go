package files

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSanitize(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"replaces unsafe characters with a dash", `a/b\c:d*e?f"g<h>i|j`, "a-b-c-d-e-f-g-h-i-j"},
		{"collapses spaces and trims", "  two   spaces \t", "two spaces"},
		{"collapses runs of replaced characters", "a//::b", "a-b"},
		{"keeps real dashes", "a--b", "a--b"},
		{"replaces control characters", "a\x00b\x1fc", "a-b-c"},
		{"turns an empty title into untitled", "", "untitled"},
		{"turns a blank title into untitled", "  \t ", "untitled"},
		{"keeps emoji", "🎙️ Episode 12 / Part 2", "🎙️ Episode 12 - Part 2"},
		{"keeps right-to-left text", "مرحبا بالعالم / שלום", "مرحبا بالعالم - שלום"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Sanitize(c.in); got != c.want {
				t.Fatalf("Sanitize(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}

	t.Run("cuts a long title to 120 bytes on a character boundary", func(t *testing.T) {
		got := Sanitize(strings.Repeat("é", 150))
		if len(got) > MaxTitleBytes {
			t.Fatalf("got %d bytes, valid=%v", len(got), utf8.ValidString(got))
		}

		if !utf8.ValidString(got) {
			t.Fatalf("got %d bytes, valid=%v", len(got), utf8.ValidString(got))
		}

		if got != strings.Repeat("é", 60) {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("cuts 4-byte characters without splitting them", func(t *testing.T) {
		got := Sanitize(strings.Repeat("🎙", 100))
		if len(got) != 120 {
			t.Fatalf("got %d bytes, valid=%v", len(got), utf8.ValidString(got))
		}

		if !utf8.ValidString(got) {
			t.Fatalf("got %d bytes, valid=%v", len(got), utf8.ValidString(got))
		}
	})

	t.Run("trims the space left at the cut", func(t *testing.T) {
		got := Sanitize(strings.Repeat("a", 119) + " bbb")
		if got != strings.Repeat("a", 119) {
			t.Fatalf("got %q", got)
		}
	})
}
