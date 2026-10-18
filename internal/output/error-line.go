package output

import (
	"errors"
	"fmt"
	"strings"

	"github.com/transcript1-com/cli/internal/api"
)

func ErrorLine(input string, err error) string {
	prefix := "transcript1: " + input + ": "
	if err == nil {
		return prefix + "failed"
	}

	var apiErr *api.Error
	if !errors.As(err, &apiErr) {
		return prefix + err.Error()
	}

	step := NextStep(apiErr)
	detail := buildErrorDetail(apiErr)
	if detail == "" {
		return prefix + apiErr.Code + ": " + step
	}

	return prefix + apiErr.Code + ": " + detail + ". " + step
}

func addHint(line, hint string) string {
	if hint == "" {
		return line
	}

	return line + "; " + hint
}

func buildErrorDetail(e *api.Error) string {
	if e.Code == api.CodeInsufficientMinutes {
		if detail := buildMinutesDetail(e); detail != "" {
			return detail
		}
	}

	return strings.TrimRight(strings.TrimSpace(e.Message), ".")
}

func buildMinutesDetail(e *api.Error) string {
	needed, hasNeeded := e.Details["minutes_needed"].(float64)
	remaining, hasRemaining := e.Details["minutes_remaining"].(float64)
	if !hasRemaining {
		return ""
	}

	if !hasNeeded {
		return fmt.Sprintf("%.0f minutes remaining", remaining)
	}

	return fmt.Sprintf("%.0f minutes needed, %.0f remaining", needed, remaining)
}
