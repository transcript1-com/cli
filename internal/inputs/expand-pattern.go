package inputs

import (
	"fmt"
	"path/filepath"
	"strings"
)

func hasPatternChars(raw string) bool {
	return strings.ContainsAny(raw, "*?[")
}

func expandPattern(raw string) ([]Input, error) {
	matches, err := filepath.Glob(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: invalid file pattern", raw)
	}

	wantsHidden := strings.HasPrefix(filepath.Base(raw), ".")
	var found []Input
	for _, match := range matches {
		if !wantsHidden && strings.HasPrefix(filepath.Base(match), ".") {
			continue
		}

		ins, err := classifyPath(match)
		if err != nil {
			return nil, err
		}

		found = append(found, ins...)
	}

	if len(found) == 0 {
		return nil, fmt.Errorf("no files match %s", raw)
	}

	return found, nil
}
