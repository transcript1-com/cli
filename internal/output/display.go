package output

import (
	"path/filepath"

	"github.com/transcript1-com/cli/internal/inputs"
)

func Display(in inputs.Input) string {
	if in.Kind == inputs.Link {
		return in.Raw
	}

	if in.Name != "" {
		return in.Name
	}

	return filepath.Base(in.Raw)
}
