package whizlabs

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

// sensitive JSON field names, compared case-insensitively. Any field
// whose key matches one of these is replaced with "<redacted>" before
// the body is handed to a logger.
var redactFields = []string{ //nolint:gochecknoglobals // static lookup table
	"password",
	"access_token",
	"refresh_token",
	"auth_token",
	"user_token",
	"accesskey",
	"secretkey",
}

// redactHeaders returns a copy of h with sensitive values replaced.
// Only the first value of each header is kept, because that is what
// humans care about when reading debug logs.
func redactHeaders(h http.Header) map[string]string {
	out := make(map[string]string, len(h))
	for k, v := range h {
		if len(v) == 0 {
			continue
		}
		if strings.EqualFold(k, "Authorization") {
			out[k] = "Bearer <redacted>"
			continue
		}
		out[k] = v[0]
	}
	return out
}

// redactBody parses body as JSON and replaces known sensitive fields
// with "<redacted>". If body is not JSON it is returned unchanged so
// that non-JSON error pages still show up in debug logs.
//
// HTML escaping is disabled on the output encoder so debug logs show
// literal < and > characters for the redaction marker, which is much
// easier to scan than "\u003credacted\u003e".
func redactBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	var obj any
	if err := json.Unmarshal(body, &obj); err != nil {
		return string(body)
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(redactValue(obj)); err != nil {
		return string(body)
	}
	return strings.TrimRight(buf.String(), "\n")
}

func redactValue(v any) any {
	switch vv := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(vv))
		for k, val := range vv {
			if isSensitiveField(k) {
				out[k] = "<redacted>"
				continue
			}
			out[k] = redactValue(val)
		}
		return out
	case []any:
		out := make([]any, len(vv))
		for i, el := range vv {
			out[i] = redactValue(el)
		}
		return out
	default:
		return v
	}
}

func isSensitiveField(name string) bool {
	for _, f := range redactFields {
		if strings.EqualFold(name, f) {
			return true
		}
	}
	return false
}
