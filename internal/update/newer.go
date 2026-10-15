package update

import (
	"strconv"
	"strings"
)

func Newer(latest, current string) bool {
	l, ok := parseVersion(latest)
	if !ok {
		return false
	}

	c, ok := parseVersion(current)
	if !ok {
		return false
	}

	for i := range l {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}

	return false
}

func IsVersion(v string) bool {
	_, ok := parseVersion(v)
	return ok
}

func isDigits(part string) bool {
	if part == "" {
		return false
	}

	return strings.Trim(part, "0123456789") == ""
}

func parseVersion(v string) ([3]int, bool) {
	var out [3]int
	parts := strings.Split(v, ".")
	if len(parts) != len(out) {
		return out, false
	}

	for i, part := range parts {
		if !isDigits(part) {
			return out, false
		}

		n, err := strconv.Atoi(part)
		if err != nil {
			return out, false
		}

		out[i] = n
	}

	return out, true
}
