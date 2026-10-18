package inputs

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

func classify(raw string) ([]Input, error) {
	if strings.Contains(raw, "://") {
		in, err := newLink(raw)
		return []Input{in}, err
	}

	_, err := os.Stat(raw)
	if errors.Is(err, fs.ErrNotExist) {
		if hasPatternChars(raw) {
			return expandPattern(raw)
		}
	}

	return classifyPath(raw)
}

func classifyPath(raw string) ([]Input, error) {
	info, err := os.Stat(raw)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%s: no such file or folder", raw)
	}

	if err != nil {
		return nil, fmt.Errorf("%s: %w", raw, unwrapPath(err))
	}

	if info.IsDir() {
		return expandFolder(raw)
	}

	in, err := newFile(raw, info)

	return []Input{in}, err
}

func unwrapPath(err error) error {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return pe.Err
	}

	return err
}
