package files

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	MaxTitleBytes = 120
	untitled      = "untitled"
	unsafeChars   = `/\:*?"<>|`
)

func Sanitize(s string) string {
	s = strings.TrimSpace(cut(clean(s), MaxTitleBytes))
	if slices.Contains([]string{"", ".", ".."}, s) {
		return untitled
	}

	return s
}

func isUnsafe(r rune) bool {
	if unicode.IsControl(r) {
		return true
	}

	return strings.ContainsRune(unsafeChars, r)
}

func clean(s string) string {
	var b strings.Builder
	lastSpace := false
	lastDash := false
	for _, r := range s {
		switch {
		case unicode.IsSpace(r):
			if !lastSpace {
				b.WriteByte(' ')
			}

			lastSpace = true
			lastDash = false
		case isUnsafe(r):
			if !lastDash {
				b.WriteByte('-')
			}

			lastSpace = false
			lastDash = true
		default:
			b.WriteRune(r)
			lastSpace = false
			lastDash = false
		}
	}

	return strings.TrimSpace(b.String())
}

func cut(s string, max int) string {
	if len(s) <= max {
		return s
	}

	for n := max; n > 0; n-- {
		if utf8.RuneStart(s[n]) {
			return s[:n]
		}
	}

	return ""
}
