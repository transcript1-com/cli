package output

import (
	"strconv"
	"strings"

	"github.com/transcript1/cli/internal/batch"
)

func buildFinalText(res batch.Result) (text, color string) {
	switch res.Status {
	case batch.Done:
		return buildDoneText(res), green
	case batch.Skipped:
		if res.File == "" {
			return "skipped (already exists)", ""
		}

		return "skipped (already exists: " + res.File + ")", ""
	case batch.SkippedMinutes:
		return "skipped (out of minutes)", ""
	case batch.StillRunning:
		if res.T == nil {
			return "still running", yellow
		}

		if res.T.ID == "" {
			return "still running", yellow
		}

		return "still running (" + res.T.ID + ")", yellow
	case batch.Interrupted:
		return "interrupted", yellow
	}

	return "", ""
}

func isReused(res batch.Result) bool {
	if res.T == nil {
		return false
	}

	return res.T.Reused
}

func buildCostText(res batch.Result) string {
	if isReused(res) {
		return "reused"
	}

	return strconv.Itoa(getBilledMinutes(res)) + " min"
}

func buildDoneText(res batch.Result) string {
	parts := []string{"done", buildCostText(res)}

	if res.File != "" {
		parts = append(parts, res.File)
	}

	return strings.Join(parts, " · ")
}

func getBilledMinutes(res batch.Result) int {
	if res.T == nil {
		return 0
	}

	if res.T.Reused {
		return 0
	}

	if res.T.BilledMinutes == nil {
		return 0
	}

	return *res.T.BilledMinutes
}
