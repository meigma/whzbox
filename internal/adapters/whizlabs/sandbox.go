package whizlabs

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/meigma/whzbox/internal/core/sandbox"
	"github.com/meigma/whzbox/internal/core/session"
)

// exchangeForPlayToken swaps a main-API JWT for a play.whizlabs.com-
// scoped token. The play token is used for all sandbox operations and
// has a separate lifetime from the main JWT; it is not persisted and is
// re-derived at the start of every sandbox method call.
//
// This matches the Whizlabs browser SPA, which also re-fetches the
// play token on every sandbox operation.
func (c *Client) exchangeForPlayToken(ctx context.Context, t session.Tokens) (string, error) {
	body := map[string]any{
		"user_token": t.AccessToken,
		"pt":         1,
	}
	type data struct {
		AuthToken string `json:"auth_token"`
	}
	var env envelope[data]
	if err := c.postJSON(ctx, c.playURL+"/api/web/login/user-authentication", body, &env); err != nil {
		return "", fmt.Errorf("exchange play token: %w", err)
	}
	if !env.Status || env.Data.AuthToken == "" {
		return "", fmt.Errorf("exchange play token: %s", env.Message)
	}
	return env.Data.AuthToken, nil
}

// playCreateData is the successful-response body for play-create-sandbox.
// Field names match the upstream JSON exactly.
type playCreateData struct {
	LoginLink    string `json:"login_link"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	AccessKey    string `json:"accesskey"`
	SecretKey    string `json:"secretkey"`
	UniqueRoleID string `json:"UniqueRoleId"`
	StartTime    string `json:"start_time"`
	EndTime      string `json:"end_time"`
}

// Create implements sandbox.Manager.
func (c *Client) Create(
	ctx context.Context,
	tokens session.Tokens,
	slug string,
	duration time.Duration,
) (*sandbox.Sandbox, error) {
	hours, err := durationToHours(duration)
	if err != nil {
		return nil, err
	}

	playJWT, err := c.exchangeForPlayToken(ctx, tokens)
	if err != nil {
		return nil, err
	}

	req := map[string]any{
		"duration":     strconv.Itoa(hours),
		"access_token": playJWT,
		"sandbox_slug": slug,
		"pt":           1,
	}
	var env envelope[playCreateData]
	err = c.postJSON(
		ctx,
		c.playURL+"/api/web/play-sandbox/play-create-sandbox",
		req,
		&env,
		withBearer(playJWT),
	)
	if err != nil {
		return nil, err
	}
	if !env.Status || env.Data.AccessKey == "" {
		return nil, fmt.Errorf("create sandbox: %s", env.Message)
	}

	return &sandbox.Sandbox{
		Slug: slug,
		Credentials: sandbox.Credentials{
			AccessKey: env.Data.AccessKey,
			SecretKey: env.Data.SecretKey,
		},
		Console: sandbox.Console{
			URL:      env.Data.LoginLink,
			Username: env.Data.Username,
			Password: env.Data.Password,
		},
		StartedAt: parseWhizlabsTime(env.Data.StartTime),
		ExpiresAt: parseWhizlabsTime(env.Data.EndTime),
	}, nil
}

// Commit implements sandbox.Manager by calling play-update-sandbox.
// The upstream update call registers ownership of the sandbox with
// the user account; without it, play-end-sandbox later returns
// "Sandbox Not Found". This two-phase dance was discovered during
// feasibility testing.
func (c *Client) Commit(ctx context.Context, tokens session.Tokens, slug string, duration time.Duration) error {
	hours, err := durationToHours(duration)
	if err != nil {
		return err
	}

	playJWT, err := c.exchangeForPlayToken(ctx, tokens)
	if err != nil {
		return err
	}

	req := map[string]any{
		"duration":     strconv.Itoa(hours),
		"access_token": playJWT,
		"sandbox_slug": slug,
	}
	var env envelope[json.RawMessage]
	err = c.postJSON(
		ctx,
		c.playURL+"/api/web/play-sandbox/play-update-sandbox",
		req,
		&env,
		withBearer(playJWT),
	)
	if err != nil {
		return err
	}
	if !env.Status {
		return fmt.Errorf("commit sandbox: %s", env.Message)
	}
	return nil
}

// Destroy implements sandbox.Manager. The upstream API identifies the
// sandbox to tear down from the session, so no sandbox ID is needed.
//
// A "Sandbox Not Found" response is translated to ErrNoActiveSandbox
// so callers can distinguish "no-op" from real failures.
func (c *Client) Destroy(ctx context.Context, tokens session.Tokens) error {
	playJWT, err := c.exchangeForPlayToken(ctx, tokens)
	if err != nil {
		return err
	}

	req := map[string]any{
		"error_id":     "0",
		"type":         "stop-sandbox",
		"access_token": playJWT,
	}
	var env envelope[json.RawMessage]
	err = c.postJSON(
		ctx,
		c.playURL+"/api/web/play-sandbox/play-end-sandbox",
		req,
		&env,
		withBearer(playJWT),
	)
	if err != nil {
		return err
	}
	if !env.Status {
		if strings.Contains(strings.ToLower(env.Message), "sandbox not found") {
			return fmt.Errorf("%w: %s", sandbox.ErrNoActiveSandbox, env.Message)
		}
		return fmt.Errorf("destroy sandbox: %s", env.Message)
	}
	return nil
}

// Active implements sandbox.Manager.
//
// The Whizlabs API does not expose a clean endpoint for querying the
// currently-active sandbox — the play-get-aws-sandbox-content endpoint
// returns a static description of allowed services, not runtime state.
// Until a proper endpoint is discovered, Active returns
// ErrNoActiveSandbox, which the service layer translates to (nil, nil)
// for `whzbox status`.
//
// Destroy does NOT depend on Active because the upstream resolves
// ownership from the session itself, so the lifecycle still works
// end-to-end despite this limitation.
func (c *Client) Active(_ context.Context, _ session.Tokens) (*sandbox.Sandbox, error) {
	return nil, sandbox.ErrNoActiveSandbox
}

// durationToHours converts a Go Duration into the integer-hours the
// Whizlabs API expects. Durations below 1h or above 9h are rejected;
// fractional hours are rounded UP (1h30m -> 2h) because the user
// presumably wanted "at least" that long.
func durationToHours(d time.Duration) (int, error) {
	if d <= 0 {
		return 0, fmt.Errorf("duration must be positive, got %v", d)
	}
	hours := int(d / time.Hour)
	if d > time.Duration(hours)*time.Hour {
		hours++
	}
	if hours < 1 || hours > 9 {
		return 0, fmt.Errorf("duration must be between 1h and 9h, got %v", d)
	}
	return hours, nil
}

// parseWhizlabsTime parses the non-standard "2006-01-02 15:04:05" times
// the play API returns. Unparseable inputs produce zero time values.
func parseWhizlabsTime(s string) time.Time {
	t, err := time.Parse("2006-01-02 15:04:05", s)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}
