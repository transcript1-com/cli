package files

import (
	"path/filepath"
	"runtime"
	"testing"
)

type targetCase struct {
	name     string
	out      string
	n        int
	wantDir  string
	wantFile string
}

func TestTarget(t *testing.T) {
	existing := t.TempDir()
	root := string(filepath.Separator)
	cases := []targetCase{
		{"prints one input to stdout", "", 1, "", ""},
		{"writes one input to a file", "out.srt", 1, "", "out.srt"},
		{"writes one input into a folder ending in a slash", "out/", 1, "out", ""},
		{"writes one input into an existing folder", existing, 1, existing, ""},
		{"defaults several inputs to transcripts", "", 3, "transcripts", ""},
		{"writes several inputs into the given folder", "x", 3, "x", ""},
		{"writes several inputs into a folder with a dot", "out.d", 3, "out.d", ""},
		{"drops the trailing slash for several inputs", "out/", 3, "out", ""},
		{"keeps the root for one input", root, 1, root, ""},
		{"keeps the root for several inputs", root, 3, root, ""},
	}
	if runtime.GOOS == "windows" {
		cases = append(cases,
			targetCase{"keeps the drive root for one input", `D:\`, 1, `D:\`, ""},
			targetCase{"keeps the drive root for several inputs", `D:\`, 3, `D:\`, ""},
		)
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
