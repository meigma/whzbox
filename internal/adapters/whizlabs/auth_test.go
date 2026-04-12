package whizlabs_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/meigma/whzbox/internal/adapters/whizlabs"
	"github.com/meigma/whzbox/internal/config"
	"github.com/meigma/whzbox/internal/core/session"
)

// makeJWT builds a three-segment JWT with the supplied claims. The
// signature is an unused placeholder — the client code never verifies
// signatures, it just reads the payload.
func makeJWT(exp int64, email string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	claims := map[string]any{"exp": exp, "user_email": email}
	payloadJSON, _ := json.Marshal(claims)
	payload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	return header + "." + payload + ".sig"
}

func loginResponse(access, refresh string) string {
	return fmt.Sprintf(
		`{"success":1,"statusCode":200,"message":"Login successful","data":{"access_token":"%s","refresh_token":"%s"}}`,
		access, refresh,
	)
}

func TestClient_Login_HappyPath(t *testing.T) {
	var captured struct {
		method string
		path   string
		body   []byte
		auth   string
		sid    string
	}

	far := time.Now().Add(24 * time.Hour).Unix()
	access := makeJWT(far, "alice@example.com")
	refresh := makeJWT(time.Now().Add(7*24*time.Hour).Unix(), "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.method = r.Method
		captured.path = r.URL.Path
		captured.auth = r.Header.Get("Authorization")
		captured.sid = r.Header.Get("X-Session-Id")
		captured.body, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, loginResponse(access, refresh))
	}))
	defer srv.Close()

	c := whizlabs.NewClient(config.WhizlabsConfig{BaseURL: srv.URL, PlayURL: srv.URL}, nil)

	tokens, err := c.Login(context.Background(), "alice@example.com", "hunter2")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	if captured.method != http.MethodPost {
		t.Errorf("method: got %q, want POST", captured.method)
	}
	if captured.path != "/Stage/auth/login" {
		t.Errorf("path: got %q", captured.path)
	}
	if captured.auth != "" {
		t.Errorf("Authorization must be empty on login, got %q", captured.auth)
	}
	if captured.sid == "" {
		t.Error("X-Session-Id must be set")
	}
	body := string(captured.body)
	for _, want := range []string{"alice@example.com", "hunter2", captured.sid, `"device_type":"desktop"`} {
		if !strings.Contains(body, want) {
			t.Errorf("request body missing %q: %s", want, body)
		}
	}

	if tokens.AccessToken != access {
		t.Errorf("AccessToken: got %q, want %q", tokens.AccessToken, access)
	}
	if tokens.RefreshToken != refresh {
		t.Errorf("RefreshToken mismatch")
	}
	if tokens.UserEmail != "alice@example.com" {
		t.Errorf("UserEmail: got %q, want alice@example.com", tokens.UserEmail)
	}
	// JWT exp is unix seconds; assert we parsed it back within a second.
	if diff := tokens.AccessTokenExpiresAt.Unix() - far; diff != 0 {
		t.Errorf("AccessTokenExpiresAt drift: got %d, want %d", tokens.AccessTokenExpiresAt.Unix(), far)
	}
}

func TestClient_Login_401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"success":0,"message":"invalid credentials"}`)
	}))
	defer srv.Close()

	c := whizlabs.NewClient(config.WhizlabsConfig{BaseURL: srv.URL}, nil)
	_, err := c.Login(context.Background(), "bad", "bad")
	if !errors.Is(err, session.ErrInvalidCredentials) {
		t.Errorf("error: got %v, want ErrInvalidCredentials", err)
	}
}

func TestClient_Login_SuccessZero(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// HTTP 200 but the envelope says the login failed.
		_, _ = io.WriteString(w, `{"success":0,"message":"User not found"}`)
	}))
	defer srv.Close()

	c := whizlabs.NewClient(config.WhizlabsConfig{BaseURL: srv.URL}, nil)
	_, err := c.Login(context.Background(), "x", "y")
	if !errors.Is(err, session.ErrInvalidCredentials) {
		t.Errorf("error: got %v, want ErrInvalidCredentials", err)
	}
}

func TestClient_Refresh_HappyPath(t *testing.T) {
	var captured struct {
		method string
		path   string
		auth   string
	}

	far := time.Now().Add(24 * time.Hour).Unix()
	access := makeJWT(far, "alice@example.com")
	refresh := makeJWT(time.Now().Add(7*24*time.Hour).Unix(), "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.method = r.Method
		captured.path = r.URL.Path
		captured.auth = r.Header.Get("Authorization")
		_, _ = io.WriteString(w, loginResponse(access, refresh))
	}))
	defer srv.Close()

	c := whizlabs.NewClient(config.WhizlabsConfig{BaseURL: srv.URL}, nil)

	current := session.Tokens{AccessToken: "old-access", UserEmail: "alice@example.com"}
	tokens, err := c.Refresh(context.Background(), current)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	if captured.method != http.MethodGet {
		t.Errorf("method: got %q, want GET", captured.method)
	}
	if captured.path != "/Stage/auth/exchange" {
		t.Errorf("path: got %q", captured.path)
	}
	if captured.auth != "Bearer old-access" {
		t.Errorf("auth header: got %q, want %q", captured.auth, "Bearer old-access")
	}
	if tokens.AccessToken != access {
		t.Errorf("AccessToken mismatch")
	}
}

func TestClient_Refresh_401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"success":0,"message":"expired"}`)
	}))
	defer srv.Close()

	c := whizlabs.NewClient(config.WhizlabsConfig{BaseURL: srv.URL}, nil)
	_, err := c.Refresh(context.Background(), session.Tokens{AccessToken: "old"})
	if !errors.Is(err, session.ErrSessionExpired) {
		t.Errorf("error: got %v, want ErrSessionExpired", err)
	}
}

func TestClient_Refresh_SuccessZero(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"success":0,"message":"cannot refresh"}`)
	}))
	defer srv.Close()

	c := whizlabs.NewClient(config.WhizlabsConfig{BaseURL: srv.URL}, nil)
	_, err := c.Refresh(context.Background(), session.Tokens{AccessToken: "old"})
	if !errors.Is(err, session.ErrSessionExpired) {
		t.Errorf("error: got %v, want ErrSessionExpired", err)
	}
}

func TestClient_Login_ContextCanceled(t *testing.T) {
	blocked := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
		close(blocked)
	}))
	defer srv.Close()

	c := whizlabs.NewClient(config.WhizlabsConfig{BaseURL: srv.URL}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.Login(ctx, "x", "y")
	if err == nil {
		t.Fatal("expected error from canceled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("error: got %v, want wrapped context.Canceled", err)
	}
}

func TestClient_Login_UnparsableJWT_UsesFallbacks(t *testing.T) {
	// Server returns non-JWT tokens; client should still return a
	// Tokens value with fallback expiries and the supplied email.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"success":1,"data":{"access_token":"garbage","refresh_token":"also-garbage"}}`)
	}))
	defer srv.Close()

	c := whizlabs.NewClient(config.WhizlabsConfig{BaseURL: srv.URL}, nil)
	tokens, err := c.Login(context.Background(), "alice@example.com", "pw")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if tokens.UserEmail != "alice@example.com" {
		t.Errorf("UserEmail fallback: got %q, want %q", tokens.UserEmail, "alice@example.com")
	}
	if tokens.AccessTokenExpiresAt.IsZero() {
		t.Error("AccessTokenExpiresAt fallback should be non-zero")
	}
	if tokens.RefreshTokenExpiresAt.IsZero() {
		t.Error("RefreshTokenExpiresAt fallback should be non-zero")
	}
}
