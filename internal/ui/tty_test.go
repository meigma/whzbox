package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsInteractive(t *testing.T) {
	t.Run("nil file", func(t *testing.T) {
		assert.False(t, IsInteractive(nil))
	})

	t.Run("file descriptor too large for term", func(t *testing.T) {
		assert.False(t, isInteractiveFD(^uintptr(0)))
	})
}
