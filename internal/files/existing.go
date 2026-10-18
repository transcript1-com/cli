package files

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/transcript1-com/cli/internal/inputs"
)

func Existing(dir string, in inputs.Input, format string, fullName bool) (string, bool) {
	if in.Kind == inputs.Link {
		if in.VideoID == "" {
			return "", false
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", false
	}

	exact := Name(in, "", format, fullName)
	suffix := " [" + Sanitize(in.VideoID) + "]" + Ext(format)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		name := e.Name()
		match := name == exact
		if in.Kind == inputs.Link {
			match = strings.HasSuffix(name, suffix)
		}

		if match {
			return filepath.Join(dir, name), true
		}
	}

	return "", false
}
