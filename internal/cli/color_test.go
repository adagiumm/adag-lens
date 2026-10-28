package cli

import (
	"bytes"
	"os"
	"testing"
)

func TestUseColor(t *testing.T) {
	t.Run("not a terminal", func(t *testing.T) {
		if useColor(&bytes.Buffer{}) {
			t.Error("a buffer is not a terminal, color must be off")
		}
	})

	t.Run("NO_COLOR wins", func(t *testing.T) {
		t.Setenv("NO_COLOR", "1")
		if useColor(os.Stdout) {
			t.Error("NO_COLOR is set, color must be off")
		}
	})

	t.Run("TERM=dumb", func(t *testing.T) {
		t.Setenv("NO_COLOR", "")
		t.Setenv("TERM", "dumb")
		if useColor(os.Stdout) {
			t.Error("TERM=dumb, color must be off")
		}
	})
}
