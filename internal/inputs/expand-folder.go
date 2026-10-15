package inputs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func expandFolder(raw string) ([]Input, error) {
	entries, err := os.ReadDir(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", raw, unwrapPath(err))
	}

	var found []Input
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}

		if MimeType(e.Name()) == "" {
			continue
		}

		p := filepath.Join(raw, e.Name())
		info, err := os.Stat(p)
		if err != nil {
			continue
		}

		if info.IsDir() {
			continue
		}

		in, err := newFile(p, info)
		if err != nil {
			return nil, err
		}

		found = append(found, in)
	}

	if len(found) == 0 {
		return nil, fmt.Errorf("%s: no audio or video files in this folder", raw)
	}

	return found, nil
}
