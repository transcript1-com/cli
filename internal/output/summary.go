package output

import (
	"fmt"
	"strings"

	"github.com/transcript1/cli/internal/batch"
)

func buildSummaryLine(results []batch.Result) string {
	var done, failed, skipped, running, used int
	for _, res := range results {
		switch res.Status {
		case batch.Done:
			done++
			used += getBilledMinutes(res)
		case batch.Started:
			done++
		case batch.Failed, batch.Interrupted:
			failed++
		case batch.Skipped, batch.SkippedMinutes:
			skipped++
		case batch.StillRunning:
			running++
		}
	}

	parts := []string{fmt.Sprintf("%d done", done)}
	counts := []struct {
		count int
		label string
	}{
		{failed, "failed"},
		{skipped, "skipped"},
		{running, "still running"},
	}
	for _, part := range counts {
		if part.count > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", part.count, part.label))
		}
	}

	return strings.Join(parts, ", ") + " · " + buildMinutesText(used) + " used"
}

func buildMinutesText(n int) string {
	if n == 1 {
		return "1 minute"
	}

	return fmt.Sprintf("%d minutes", n)
}
