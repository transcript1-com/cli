package files

import (
	"path/filepath"
	"strings"

	"github.com/transcript1-com/cli/internal/inputs"
)

func Name(in inputs.Input, title, format string, fullName bool) string {
	ext := Ext(format)
	if in.Kind == inputs.Link {
		return Sanitize(title) + " [" + Sanitize(in.VideoID) + "]" + ext
	}

	if fullName {
		return in.Name + ext
	}

	return strings.TrimSuffix(in.Name, filepath.Ext(in.Name)) + ext
}
