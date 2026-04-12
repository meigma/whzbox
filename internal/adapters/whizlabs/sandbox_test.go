package whizlabs_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/meigma/whzbox/internal/adapters/whizlabs"
	"github.com/meigma/whzbox/internal/config"
	"github.com/meigma/whzbox/internal/core/sandbox"
	"github.com/meigma/whzbox/internal/core/session"
)

// playRouter stands in for play.whizlabs.com. It records every call by
// path and returns fixed responses per route so individual tests can
// assert the specific arguments whizlabs.Client sends.
type playRouter struct {
	playToken string

	createResp []byte
	updateResp []byte
	endResp    []byte

	calls   []string
	bodies  map[string][]byte
	headers map[string]http.Header
}

func newPlayRouter() *playRouter {
	return &playRouter{
		playToken: "play-jwt-xyz",
		bodies:    map[string][]byte{},
		headers:   map[string]http.Header{},
	}
}

func (r *playRouter) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.calls = append(r.calls, req.URL.Path)
		body, _ := io.ReadAll(req.Body)
		r.bodies[req.URL.Path] = body
		r.headers[req.URL.Path] = req.Header.Clone()

		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case "/api/web/login/user-authentication":
			_, _ = io.WriteString(w, `{"status":true,"data":{"auth_token":"`+r.playToken+`"}}`)
		case "/api/web/play-sandbox/play-create-sandbox":
			_, _ = w.Write(r.createResp)
		case "/api/web/play-sandbox/play-update-sandbox":
			_, _ = w.Write(r.updateResp)
		case "/api/web/play-sandbox/play-end-sandbox":
			_, _ = w.Write(r.endResp)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
}

func (r *playRouter) bodyAt(path string) map[string]any {
	var m map[string]any
	_ = json.Unmarshal(r.bodies[path], &m)
	return m
}

func (r *playRouter) countCalls(path string) int {
	n := 0
	for _, p := range r.calls {
		if p == path {
			n++
		}
	}
	return n
}

func defaultCreateResp() []byte {
	return []byte(`{
		"status": true,
		"message": "ok",
		"data": {
			"login_link": "https://111111111111.signin.aws.amazon.com/console?region=us-east-1",
			"username": "Whiz_User_test.1",
			"password": "pw-uuid-1",
			"accesskey": "AKIA...TEST1",
			"secretkey": "secret-1",
			"start_time": "2026-04-11 12:00:00",
			"end_time": "2026-04-11 13:00:00"
		}
	}`)
}

func defaultUpdateResp() []byte {
	return []byte(`{"status":true,"message":"updated","data":null}`)
}

func defaultEndResp() []byte {
	return []byte(`{"status":true,"message":"destroyed","data":null}`)
}

func newClient(baseURL string) *whizlabs.Client {
	return whizlabs.NewClient(config.WhizlabsConfig{
		BaseURL: baseURL,
		PlayURL: baseURL,
	}, nil)
}

func sampleTokens() session.Tokens {
	return session.Tokens{AccessToken: "main-jwt", UserEmail: "alice@example.com"}
}

// -----------------------------------------------------------------------------
// Create
// -----------------------------------------------------------------------------

func TestClient_Create_HappyPath(t *testing.T) {
	router := newPlayRouter()
	router.createResp = defaultCreateResp()

	srv := httptest.NewServer(router.handler())
	defer srv.Close()

	c := newClient(srv.URL)

	sb, err := c.Create(context.Background(), sampleTokens(), "aws-sandbox", time.Hour)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if sb == nil {
		t.Fatal("Create returned nil sandbox")
	}

	// Play token exchange happened first.
	if router.countCalls("/api/web/login/user-authentication") != 1 {
		t.Errorf("expected 1 exchange call, got %d", router.countCalls("/api/web/login/user-authentication"))
	}
	// Then play-create-sandbox.
	if router.countCalls("/api/web/play-sandbox/play-create-sandbox") != 1 {
		t.Errorf("expected 1 create call")
	}

	// Create request carried the exchanged play token as Bearer.
	h := router.headers["/api/web/play-sandbox/play-create-sandbox"]
	if got := h.Get("Authorization"); got != "Bearer "+router.playToken {
		t.Errorf("Authorization: got %q, want %q", got, "Bearer "+router.playToken)
	}

	// Request body has the right shape.
	body := router.bodyAt("/api/web/play-sandbox/play-create-sandbox")
	if body["sandbox_slug"] != "aws-sandbox" {
		t.Errorf("sandbox_slug: got %v", body["sandbox_slug"])
	}
	if body["duration"] != "1" {
		t.Errorf("duration: got %v, want %q", body["duration"], "1")
	}
	if body["access_token"] != router.playToken {
		t.Errorf("access_token should be play JWT, got %v", body["access_token"])
	}

	// Response was parsed into a Sandbox value.
	if sb.Slug != "aws-sandbox" {
		t.Errorf("Slug: got %q", sb.Slug)
	}
	if sb.Credentials.AccessKey != "AKIA...TEST1" {
		t.Errorf("AccessKey: got %q", sb.Credentials.AccessKey)
	}
	if sb.Credentials.SecretKey != "secret-1" {
		t.Errorf("SecretKey mismatch")
	}
	if sb.Console.Username != "Whiz_User_test.1" {
		t.Errorf("Console.Username: got %q", sb.Console.Username)
	}
	if sb.StartedAt.IsZero() || sb.ExpiresAt.IsZero() {
		t.Errorf("times not parsed: start=%v end=%v", sb.StartedAt, sb.ExpiresAt)
	}
}

func TestClient_Create_DurationRoundedUp(t *testing.T) {
	router := newPlayRouter()
	router.createResp = defaultCreateResp()
	srv := httptest.NewServer(router.handler())
	defer srv.Close()
	c := newClient(srv.URL)

	if _, err := c.Create(context.Background(), sampleTokens(), "aws-sandbox", 90*time.Minute); err != nil {
		t.Fatalf("Create: %v", err)
	}
	body := router.bodyAt("/api/web/play-sandbox/play-create-sandbox")
	if body["duration"] != "2" {
		t.Errorf("90m should round up to 2h, got %v", body["duration"])
	}
}

func TestClient_Create_DurationValidation(t *testing.T) {
	srv := httptest.NewServer(newPlayRouter().handler())
	defer srv.Close()
	c := newClient(srv.URL)

	tests := []time.Duration{
		0,
		-time.Second,
		10 * time.Hour,
		24 * time.Hour,
	}
	for _, d := range tests {
		if _, err := c.Create(context.Background(), sampleTokens(), "aws-sandbox", d); err == nil {
			t.Errorf("duration %v should be rejected", d)
		}
	}
}

func TestClient_Create_PlayAPIStatusFalse(t *testing.T) {
	router := newPlayRouter()
	router.createResp = []byte(`{"status":false,"message":"quota exceeded"}`)
	srv := httptest.NewServer(router.handler())
	defer srv.Close()
	c := newClient(srv.URL)

	_, err := c.Create(context.Background(), sampleTokens(), "aws-sandbox", time.Hour)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "quota exceeded") {
		t.Errorf("error should carry upstream message: %v", err)
	}
}

func TestClient_Create_InvalidTimesUseFallbackWindow(t *testing.T) {
	router := newPlayRouter()
	router.createResp = []byte(`{
		"status": true,
		"message": "ok",
		"data": {
			"login_link": "https://111111111111.signin.aws.amazon.com/console?region=us-east-1",
			"username": "Whiz_User_test.1",
			"password": "pw-uuid-1",
			"accesskey": "AKIA...TEST1",
			"secretkey": "secret-1",
			"start_time": "not-a-time",
			"end_time": "also-not-a-time"
		}
	}`)
	srv := httptest.NewServer(router.handler())
	defer srv.Close()

	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelInfo}))
	c := whizlabs.NewClient(config.WhizlabsConfig{
		BaseURL: srv.URL,
		PlayURL: srv.URL,
	}, logger)

	before := time.Now().UTC()
	sb, err := c.Create(context.Background(), sampleTokens(), "aws-sandbox", 90*time.Minute)
	after := time.Now().UTC()
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if sb.StartedAt.IsZero() || sb.ExpiresAt.IsZero() {
		t.Fatalf("fallback timestamps should be non-zero: start=%v end=%v", sb.StartedAt, sb.ExpiresAt)
	}
	if sb.StartedAt.Before(before) || sb.StartedAt.After(after.Add(time.Second)) {
		t.Errorf(
			"fallback start not based on local create time: got %v, want between %v and %v",
			sb.StartedAt,
			before,
			after,
		)
	}
	if got := sb.ExpiresAt.Sub(sb.StartedAt); got != 2*time.Hour {
		t.Errorf("fallback expiry should use rounded duration: got %v, want %v", got, 2*time.Hour)
	}
	logOutput := logs.String()
	for _, want := range []string{
		"invalid sandbox timestamps",
		"start_time=not-a-time",
		"end_time=also-not-a-time",
	} {
		if !strings.Contains(logOutput, want) {
			t.Errorf("log output missing %q: %s", want, logOutput)
		}
	}
}

func TestClient_Create_InvalidTimeOrderUsesFallbackWindow(t *testing.T) {
	router := newPlayRouter()
	router.createResp = []byte(`{
		"status": true,
		"message": "ok",
		"data": {
			"login_link": "https://111111111111.signin.aws.amazon.com/console?region=us-east-1",
			"username": "Whiz_User_test.1",
			"password": "pw-uuid-1",
			"accesskey": "AKIA...TEST1",
			"secretkey": "secret-1",
			"start_time": "2026-04-11 13:00:00",
			"end_time": "2026-04-11 12:00:00"
		}
	}`)
	srv := httptest.NewServer(router.handler())
	defer srv.Close()
	c := newClient(srv.URL)

	sb, err := c.Create(context.Background(), sampleTokens(), "aws-sandbox", time.Hour)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if sb.StartedAt.IsZero() || sb.ExpiresAt.IsZero() {
		t.Fatalf("fallback timestamps should be non-zero: start=%v end=%v", sb.StartedAt, sb.ExpiresAt)
	}
	if got := sb.ExpiresAt.Sub(sb.StartedAt); got != time.Hour {
		t.Errorf("fallback expiry should preserve requested duration: got %v, want %v", got, time.Hour)
	}
}

// -----------------------------------------------------------------------------
// Commit
// -----------------------------------------------------------------------------

func TestClient_Commit_HappyPath(t *testing.T) {
	router := newPlayRouter()
	router.updateResp = defaultUpdateResp()
	srv := httptest.NewServer(router.handler())
	defer srv.Close()
	c := newClient(srv.URL)

	if err := c.Commit(context.Background(), sampleTokens(), "aws-sandbox", time.Hour); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	if router.countCalls("/api/web/play-sandbox/play-update-sandbox") != 1 {
		t.Errorf("expected 1 update call")
	}
	// Commit does its own token exchange (not shared with Create).
	if router.countCalls("/api/web/login/user-authentication") != 1 {
		t.Errorf("expected 1 exchange call, got %d", router.countCalls("/api/web/login/user-authentication"))
	}

	body := router.bodyAt("/api/web/play-sandbox/play-update-sandbox")
	if body["sandbox_slug"] != "aws-sandbox" || body["duration"] != "1" {
		t.Errorf("bad body: %v", body)
	}
}

// -----------------------------------------------------------------------------
// Destroy
// -----------------------------------------------------------------------------

func TestClient_Destroy_HappyPath(t *testing.T) {
	router := newPlayRouter()
	router.endResp = defaultEndResp()
	srv := httptest.NewServer(router.handler())
	defer srv.Close()
	c := newClient(srv.URL)

	if err := c.Destroy(context.Background(), sampleTokens()); err != nil {
		t.Fatalf("Destroy: %v", err)
	}
	if router.countCalls("/api/web/play-sandbox/play-end-sandbox") != 1 {
		t.Errorf("expected 1 end call")
	}

	body := router.bodyAt("/api/web/play-sandbox/play-end-sandbox")
	if body["error_id"] != "0" || body["type"] != "stop-sandbox" {
		t.Errorf("bad body: %v", body)
	}
}

func TestClient_Destroy_SandboxNotFound(t *testing.T) {
	router := newPlayRouter()
	router.endResp = []byte(`{"status":false,"message":"Sandbox Not Found"}`)
	srv := httptest.NewServer(router.handler())
	defer srv.Close()
	c := newClient(srv.URL)

	err := c.Destroy(context.Background(), sampleTokens())
	if !errors.Is(err, sandbox.ErrNoActiveSandbox) {
		t.Errorf("error: got %v, want ErrNoActiveSandbox", err)
	}
}

func TestClient_Destroy_GenericFailure(t *testing.T) {
	router := newPlayRouter()
	router.endResp = []byte(`{"status":false,"message":"internal error"}`)
	srv := httptest.NewServer(router.handler())
	defer srv.Close()
	c := newClient(srv.URL)

	err := c.Destroy(context.Background(), sampleTokens())
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, sandbox.ErrNoActiveSandbox) {
		t.Error("generic failure should not match ErrNoActiveSandbox")
	}
}

// -----------------------------------------------------------------------------
// Active
// -----------------------------------------------------------------------------

func TestClient_Active_AlwaysReturnsNoActive(t *testing.T) {
	srv := httptest.NewServer(newPlayRouter().handler())
	defer srv.Close()
	c := newClient(srv.URL)

	_, err := c.Active(context.Background(), sampleTokens())
	if !errors.Is(err, sandbox.ErrNoActiveSandbox) {
		t.Errorf("Active should return ErrNoActiveSandbox until the upstream exposes a query endpoint, got %v", err)
	}
}
