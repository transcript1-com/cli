package files

import (
	"os"
	"path/filepath"
)

const defaultDir = "transcripts"

func Target(out string, n int) (dir, file string) {
	if out == "" {
		if n > 1 {
			return defaultDir, ""
		}

		return "", ""
	}

	if n > 1 {
		return filepath.Clean(out), ""
	}

	if isDir(out) {
		return filepath.Clean(out), ""
	}

	return "", out
}

func isDir(path string) bool {
	last := path[len(path)-1]
	if last == '/' {
		return true
	}

	if last == filepath.Separator {
		return true
	}

	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}
