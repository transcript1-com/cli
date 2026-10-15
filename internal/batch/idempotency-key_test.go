package batch

import (
	"regexp"
	"testing"
)

var uuidV4 = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestNewIdempotencyKey(t *testing.T) {
	a, err := newIdempotencyKey()
	if err != nil {
		t.Fatal(err)
	}

	b, _ := newIdempotencyKey()

	t.Run("is a version 4 UUID", func(t *testing.T) {
		if !uuidV4.MatchString(a) {
			t.Fatalf("key %q", a)
		}
	})

	t.Run("differs between calls", func(t *testing.T) {
		if a == b {
			t.Fatalf("keys %q %q", a, b)
		}
	})
}
