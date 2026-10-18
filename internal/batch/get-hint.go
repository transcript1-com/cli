package batch

import (
	"slices"

	"github.com/transcript1-com/cli/internal/api"
)

const hintPrefix = "fetch with transcript1 get "

func (r Result) GetHint() string {
	if r.Status != Failed {
		return ""
	}

	if r.T == nil {
		return ""
	}

	if r.T.ID == "" {
		return ""
	}

	if slices.Contains([]string{api.StatusFailed, api.StatusCanceled}, r.T.Status) {
		return ""
	}

	return hintPrefix + r.T.ID
}
