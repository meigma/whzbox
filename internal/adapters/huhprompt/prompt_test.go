package huhprompt

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/meigma/whzbox/internal/core/session"
)

func TestPrompt_InjectedNonTerminalStreamsFailFast(t *testing.T) {
	prompt := New(strings.NewReader("alice@example.com\nsecret\n"), io.Discard)
	_, _, err := prompt.Credentials(context.Background(), "")
	if !errors.Is(err, session.ErrPromptUnavailable) {
		t.Errorf("error: got %v, want ErrPromptUnavailable", err)
	}
}
