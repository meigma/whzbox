package whizlabs

import (
	"net/http"
	"strings"
	"testing"
)

func TestRedactBody_RedactsSensitiveFields(t *testing.T) {
	in := []byte(`{
		"email": "x@y.com",
		"password": "secret",
		"data": {
			"access_token": "abc",
			"refresh_token": "def",
			"user": {
				"auth_token": "ghi",
				"name": "Alice"
			}
		}
	}`)
	got := redactBody(in)

	for _, leak := range []string{"secret", "abc", "def", "ghi"} {
		if strings.Contains(got, leak) {
			t.Errorf("sensitive value %q leaked: %s", leak, got)
		}
	}
	for _, keep := range []string{"x@y.com", "Alice"} {
		if !strings.Contains(got, keep) {
			t.Errorf("non-sensitive value %q missing: %s", keep, got)
		}
	}
	// And the redaction marker is present.
	if !strings.Contains(got, "<redacted>") {
		t.Errorf("redaction marker missing: %s", got)
	}
}

func TestRedactBody_EmptyInput(t *testing.T) {
	if got := redactBody(nil); got != "" {
		t.Errorf("nil body: got %q, want empty", got)
	}
	if got := redactBody([]byte{}); got != "" {
		t.Errorf("empty body: got %q, want empty", got)
	}
}

func TestRedactBody_NonJSONPassthrough(t *testing.T) {
	in := []byte("<html>oops 502</html>")
	if got := redactBody(in); got != string(in) {
		t.Errorf("non-JSON should pass through unchanged: got %q", got)
	}
}

func TestRedactBody_ArrayOfObjects(t *testing.T) {
	in := []byte(`[{"access_token":"aaa"},{"access_token":"bbb"}]`)
	got := redactBody(in)
	if strings.Contains(got, "aaa") || strings.Contains(got, "bbb") {
		t.Errorf("array redaction leaked: %s", got)
	}
}

func TestRedactHeaders_AuthorizationRedacted(t *testing.T) {
	h := http.Header{}
	h.Set("Content-Type", "application/json")
	h.Set("Authorization", "Bearer eyJabc.def.ghi")
	h.Set("X-Session-Id", "12345")

	got := redactHeaders(h)

	if got["Authorization"] != "Bearer <redacted>" {
		t.Errorf("Authorization: got %q, want Bearer <redacted>", got["Authorization"])
	}
	if got["Content-Type"] != "application/json" {
		t.Errorf("Content-Type: got %q", got["Content-Type"])
	}
	if got["X-Session-Id"] != "12345" {
		t.Errorf("X-Session-Id should be kept as-is: got %q", got["X-Session-Id"])
	}
}

func TestIsSensitiveField(t *testing.T) {
	tests := map[string]bool{
		"password":      true,
		"PASSWORD":      true,
		"AccessToken":   false, // camelCase is not one of our matches
		"access_token":  true,
		"refresh_token": true,
		"auth_token":    true,
		"user_token":    true,
		"accesskey":     true,
		"secretkey":     true,
		"email":         false,
		"username":      false,
	}
	for in, want := range tests {
		if got := isSensitiveField(in); got != want {
			t.Errorf("isSensitiveField(%q) = %v, want %v", in, got, want)
		}
	}
}
