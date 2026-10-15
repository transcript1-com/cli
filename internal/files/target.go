package files

import (
	"os"
	"strings"
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
		return strings.TrimRight(out, `/\`), ""
	}

	if isDir(out) {
		return strings.TrimRight(out, `/\`), ""
	}

	return "", out
}

func isDir(path string) bool {
	if strings.HasSuffix(path, "/") {
		return true
	}

	if strings.HasSuffix(path, `\`) {
		return true
	}

	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}
