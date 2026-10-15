package files

import "testing"

func TestTarget(t *testing.T) {
	existing := t.TempDir()
	cases := []struct {
		name     string
		out      string
		n        int
		wantDir  string
		wantFile string
	}{
		{"prints one input to stdout", "", 1, "", ""},
		{"writes one input to a file", "out.srt", 1, "", "out.srt"},
		{"writes one input into a folder ending in a slash", "out/", 1, "out", ""},
		{"writes one input into an existing folder", existing, 1, existing, ""},
		{"defaults several inputs to transcripts", "", 3, "transcripts", ""},
		{"writes several inputs into the given folder", "x", 3, "x", ""},
		{"writes several inputs into a folder with a dot", "out.d", 3, "out.d", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir, file := Target(c.out, c.n)
			if dir != c.wantDir {
				t.Fatalf("Target(%q, %d) = %q, %q; want %q, %q", c.out, c.n, dir, file, c.wantDir, c.wantFile)
			}

			if file != c.wantFile {
				t.Fatalf("Target(%q, %d) = %q, %q; want %q, %q", c.out, c.n, dir, file, c.wantDir, c.wantFile)
			}
		})
	}
}
