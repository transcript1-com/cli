package cli

import (
	"strconv"
	"strings"
)

func needsComma(i, length int) bool {
	if i == 0 {
		return false
	}

	return (length-i)%3 == 0
}

func formatThousands(n int) string {
	digits := strconv.Itoa(n)
	sign := ""
	if n < 0 {
		sign = "-"
		digits = digits[1:]
	}

	var b strings.Builder
	for i, digit := range digits {
		if needsComma(i, len(digits)) {
			b.WriteByte(',')
		}

		b.WriteRune(digit)
	}

	return sign + b.String()
}
