package inputs

import "testing"

func TestMimeType(t *testing.T) {
	cases := map[string]string{
		"a.M4A":  "audio/mp4",
		"a.mp3":  "audio/mpeg",
		"a.wav":  "audio/wav",
		"a.flac": "audio/flac",
		"a.ogg":  "audio/ogg",
		"a.opus": "audio/opus",
		"a.mp4":  "video/mp4",
		"a.mov":  "video/quicktime",
		"a.webm": "video/webm",
		"a.mkv":  "video/x-matroska",
		"a.txt":  "",
	}
	for name, want := range cases {
		t.Run(name+" is "+want, func(t *testing.T) {
			if got := MimeType(name); got != want {
				t.Fatalf("MimeType(%q) = %q, want %q", name, got, want)
			}
		})
	}
}
