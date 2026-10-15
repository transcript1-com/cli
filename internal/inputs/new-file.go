package inputs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func newFile(raw string, info os.FileInfo) (Input, error) {
	if !info.Mode().IsRegular() {
		return Input{}, fmt.Errorf("%s: not a regular file", raw)
	}

	if MimeType(info.Name()) == "" {
		return Input{}, fmt.Errorf("%s: unsupported file type (use %s)", raw, buildExtensionList())
	}

	if info.Size() > MaxUploadBytes {
		return Input{}, fmt.Errorf("%s: larger than %s", raw, maxUploadText)
	}

	abs, err := filepath.Abs(raw)
	if err != nil {
		return Input{}, fmt.Errorf("%s: %w", raw, err)
	}

	in := Input{
		Raw:  raw,
		Kind: File,
		Path: abs,
		Name: filepath.Base(abs),
		Size: info.Size(),
	}

	return in, nil
}

func buildExtensionList() string {
	last := len(Extensions) - 1
	return strings.Join(Extensions[:last], ", ") + " or " + Extensions[last]
}
