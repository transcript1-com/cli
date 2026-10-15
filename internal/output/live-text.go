package output

import (
	"fmt"
	"unicode/utf8"

	"github.com/transcript1/cli/internal/batch"
)

func (r *Reporter) buildLiveText(e batch.Event) string {
	prefix := fmt.Sprintf("[%d/%d] ", e.Index+1, r.total)
	suffix := "  " + buildStageText(e, true)
	room := r.width - 1 - utf8.RuneCountInString(prefix) - utf8.RuneCountInString(suffix)

	return prefix + shorten(Display(e.Input), room) + suffix
}

func shorten(text string, room int) string {
	if utf8.RuneCountInString(text) <= room {
		return text
	}

	if room < 1 {
		return "…"
	}

	runes := []rune(text)

	return string(runes[:room-1]) + "…"
}
