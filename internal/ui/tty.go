package ui

import (
	"math"
	"os"

	"golang.org/x/term"
)

// IsInteractive reports whether the given file is attached to a terminal.
// A nil file always reports false.
//
// This is the predicate used by huhprompt to decide whether to launch an
// interactive form or to bail out with session.ErrPromptUnavailable, and
// by the CLI to gate destructive confirmation prompts.
func IsInteractive(f *os.File) bool {
	if f == nil {
		return false
	}
	return isInteractiveFD(f.Fd())
}

func isInteractiveFD(fd uintptr) bool {
	if fd > math.MaxInt {
		return false
	}
	return term.IsTerminal(int(fd))
}
