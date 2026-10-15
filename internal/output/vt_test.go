package output

import (
	"bytes"
	"testing"
)

func TestEnableEscapes(t *testing.T) {
	t.Run("leaves a writer that is not a console alone", func(t *testing.T) {
		if !EnableEscapes(&bytes.Buffer{}) {
			t.Fatal("refused a buffer")
		}
	})
}
