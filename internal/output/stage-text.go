package output

import (
	"fmt"

	"github.com/transcript1/cli/internal/batch"
)

func buildStageText(e batch.Event, withPercent bool) string {
	switch e.Stage {
	case batch.WaitingSlot:
		return "waiting for a free job slot"
	case batch.Uploading:
		if !withPercent {
			return "uploading"
		}

		if e.Total <= 0 {
			return "uploading"
		}

		return fmt.Sprintf("uploading %d%%", e.Sent*100/e.Total)
	case batch.Queued:
		return "queued"
	case batch.Transcribing:
		return "transcribing"
	}

	return ""
}
