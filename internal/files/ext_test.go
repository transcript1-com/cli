package files

import "testing"

func TestExt(t *testing.T) {
	cases := map[string]string{
		"text": ".txt",
		"txt":  ".txt",
		"srt":  ".srt",
		"vtt":  ".vtt",
		"json": ".json",
		"":     ".txt",
	}
	for format, want := range cases {
		t.Run("maps "+format+" to "+want, func(t *testing.T) {
			if got := Ext(format); got != want {
				t.Fatalf("Ext(%q) = %q, want %q", format, got, want)
			}
		})
	}
}
