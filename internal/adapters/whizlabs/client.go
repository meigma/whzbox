package whizlabs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/meigma/whzbox/internal/config"
)

// userAgent is the value sent in the User-Agent header. Kept as a
// constant (rather than wiring up cli.Version) to keep this package
// importable from the CLI without creating a cycle.
const userAgent = "whzbox/dev (+https://github.com/meigma/whzbox)"

// defaultTimeout is the per-request HTTP timeout. Whizlabs sandbox
// creation takes up to ~30s, so this must comfortably exceed that.
const defaultTimeout = 60 * time.Second

// maxBodyTruncate is the maximum number of characters to include in
// HTTP error messages.
const maxBodyTruncate = 200

// Client is the HTTP client for the Whizlabs API. It implements
// session.IdentityProvider and sandbox.Manager.
//
// Instances are safe for concurrent use. The underlying [http.Client] is
// shared across all calls.
type Client struct {
	baseURL string
	playURL string
	http    *http.Client
	logger  *slog.Logger
}

// NewClient returns a Client configured against the given endpoints.
//
// A nil logger is replaced with a discard handler. Callers that need a
// different HTTP timeout or transport can replace Client.http after
// construction.
func NewClient(cfg config.WhizlabsConfig, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return &Client{
		baseURL: cfg.BaseURL,
		playURL: cfg.PlayURL,
		http: &http.Client{
			Timeout: defaultTimeout,
		},
		logger: logger,
	}
}

// BaseURL returns the configured Whizlabs API base URL.
func (c *Client) BaseURL() string { return c.baseURL }

// PlayURL returns the configured play.whizlabs.com base URL.
func (c *Client) PlayURL() string { return c.playURL }

// requestOpt mutates an outgoing request (e.g. to attach auth headers).
type requestOpt func(*http.Request)

// withBearer attaches an Authorization: Bearer header if the token is
// non-empty.
func withBearer(token string) requestOpt {
	return func(r *http.Request) {
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
	}
}

// withSessionID attaches the X-Session-Id header if non-empty.
func withSessionID(id string) requestOpt {
	return func(r *http.Request) {
		if id != "" {
			r.Header.Set("X-Session-Id", id)
		}
	}
}

// postJSON marshals body to JSON, POSTs it to url, and unmarshals the
// 2xx response into out.
func (c *Client) postJSON(ctx context.Context, url string, body, out any, opts ...requestOpt) error {
	return c.doJSON(ctx, http.MethodPost, url, body, out, opts...)
}

// getJSON performs a GET and unmarshals the 2xx response into out.
func (c *Client) getJSON(ctx context.Context, url string, out any, opts ...requestOpt) error {
	return c.doJSON(ctx, http.MethodGet, url, nil, out, opts...)
}

func (c *Client) doJSON(ctx context.Context, method, url string, body, out any, opts ...requestOpt) error {
	var bodyReader io.Reader
	var rawBody []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		rawBody = b
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}

	// Standard headers that the Whizlabs API expects from browser traffic.
	// Setting them unconditionally keeps us looking like a regular web
	// client and avoids bot-detection heuristics on the API gateway.
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Origin", "https://www.whizlabs.com")
	req.Header.Set("Referer", "https://www.whizlabs.com/")
	req.Header.Set("User-Agent", userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, opt := range opts {
		opt(req)
	}

	c.debugRequest(req, rawBody)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("http %s %s: %w", method, url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	c.debugResponse(resp, respBody)

	if resp.StatusCode >= http.StatusBadRequest {
		return &HTTPError{
			Method:     method,
			URL:        url,
			StatusCode: resp.StatusCode,
			Body:       respBody,
		}
	}

	if out != nil {
		err = json.Unmarshal(respBody, out)
		if err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}
	return nil
}

// HTTPError is returned by doJSON when the upstream responds with a 4xx
// or 5xx status code. Callers can use [errors.As] to inspect the status
// code and map specific responses to their own sentinel errors.
type HTTPError struct {
	Method     string
	URL        string
	StatusCode int
	Body       []byte
}

// Error implements error.
func (e *HTTPError) Error() string {
	return fmt.Sprintf("%s %s: HTTP %d: %s", e.Method, e.URL, e.StatusCode, truncate(string(e.Body), maxBodyTruncate))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// debugRequest logs the outgoing request at debug level with sensitive
// headers and body fields redacted.
func (c *Client) debugRequest(req *http.Request, body []byte) {
	if !c.logger.Enabled(req.Context(), slog.LevelDebug) {
		return
	}
	c.logger.Debug("whizlabs request",
		"method", req.Method,
		"url", req.URL.String(),
		"headers", redactHeaders(req.Header),
		"body", redactBody(body),
	)
}

// debugResponse logs the incoming response at debug level with the body
// redacted.
func (c *Client) debugResponse(resp *http.Response, body []byte) {
	if !c.logger.Enabled(resp.Request.Context(), slog.LevelDebug) {
		return
	}
	c.logger.Debug("whizlabs response",
		"status", resp.StatusCode,
		"body", redactBody(body),
	)
}
