package whizlabs

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/meigma/whzbox/internal/core/session"
)

const (
	accessTokenFallbackTTL  = 15 * time.Hour
	refreshTokenFallbackTTL = 7 * 24 * time.Hour
	jwtSegmentCount         = 3
)

// loginReq is the JSON body of POST /Stage/auth/login. The "analytics"
// field is sent by the real browser client; the upstream seems to
// accept requests without it but we include it to stay looking normal.
type loginReq struct {
	Email     string         `json:"email"`
	Password  string         `json:"password"`
	Analytics loginAnalytics `json:"analytics"`
}

type loginAnalytics struct {
	SessionID  string `json:"session_id"`
	DeviceType string `json:"device_type"`
	App        string `json:"app"`
}

// envelope is the standard response shape from the Whizlabs auth API
// and the play.whizlabs.com sandbox API.
//
// Annoyingly, the two APIs use different success fields:
//
//   - auth API: {"success": 1 | 0}
//   - play API: {"status": true | false}
//
// The envelope carries both so one generic type works for both.
// Unused fields stay zero.
type envelope[T any] struct {
	Success    int    `json:"success"`
	Status     bool   `json:"status"`
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	Data       T      `json:"data"`
}

// tokenData is the token pair returned by /auth/login and /auth/exchange.
type tokenData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Login exchanges an email/password pair for a fresh Tokens value by
// posting to /Stage/auth/login. It implements session.IdentityProvider.
func (c *Client) Login(ctx context.Context, email, password string) (session.Tokens, error) {
	sid := newSessionID()
	req := loginReq{
		Email:    email,
		Password: password,
		Analytics: loginAnalytics{
			SessionID:  sid,
			DeviceType: "desktop",
			App:        "web",
		},
	}

	var env envelope[tokenData]
	err := c.postJSON(ctx, c.baseURL+"/Stage/auth/login", req, &env, withSessionID(sid))
	if err != nil {
		var herr *HTTPError
		if errors.As(err, &herr) && herr.StatusCode == http.StatusUnauthorized {
			return session.Tokens{}, session.ErrInvalidCredentials
		}
		return session.Tokens{}, err
	}

	if env.Success != 1 || env.Data.AccessToken == "" {
		return session.Tokens{}, fmt.Errorf("%w: %s", session.ErrInvalidCredentials, env.Message)
	}

	return tokensFromJWT(env.Data.AccessToken, env.Data.RefreshToken, email), nil
}

// Refresh exchanges a currently-valid or recently-expired token pair
// for a fresh one via GET /Stage/auth/exchange. It implements
// session.IdentityProvider.
func (c *Client) Refresh(ctx context.Context, current session.Tokens) (session.Tokens, error) {
	var env envelope[tokenData]
	err := c.getJSON(ctx, c.baseURL+"/Stage/auth/exchange", &env, withBearer(current.AccessToken))
	if err != nil {
		var herr *HTTPError
		if errors.As(err, &herr) && herr.StatusCode == http.StatusUnauthorized {
			return session.Tokens{}, fmt.Errorf("%w: HTTP 401", session.ErrSessionExpired)
		}
		return session.Tokens{}, err
	}

	if env.Success != 1 || env.Data.AccessToken == "" {
		return session.Tokens{}, fmt.Errorf("%w: %s", session.ErrSessionExpired, env.Message)
	}

	return tokensFromJWT(env.Data.AccessToken, env.Data.RefreshToken, current.UserEmail), nil
}

// tokensFromJWT decodes the 'exp' and 'user_email' claims from the
// middle segment of the supplied JWTs without verifying the signature
// (we trust the upstream because we just received them over TLS).
//
// When either token cannot be parsed, sensible fallbacks are used based
// on the prototype's observed behaviour: 15h access, 7d refresh.
func tokensFromJWT(access, refresh, fallbackEmail string) session.Tokens {
	accessExp, email := parseJWTExpiryEmail(access)
	refreshExp, _ := parseJWTExpiryEmail(refresh)

	if email == "" {
		email = fallbackEmail
	}
	if accessExp.IsZero() {
		accessExp = time.Now().Add(accessTokenFallbackTTL)
	}
	if refreshExp.IsZero() {
		refreshExp = time.Now().Add(refreshTokenFallbackTTL)
	}

	return session.Tokens{
		AccessToken:           access,
		RefreshToken:          refresh,
		AccessTokenExpiresAt:  accessExp,
		RefreshTokenExpiresAt: refreshExp,
		UserEmail:             email,
	}
}

// parseJWTExpiryEmail decodes the payload segment of a JWT as JSON and
// extracts the 'exp' and 'user_email' claims. It returns zero values
// if the token is malformed.
func parseJWTExpiryEmail(token string) (time.Time, string) {
	parts := strings.Split(token, ".")
	if len(parts) != jwtSegmentCount {
		return time.Time{}, ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		// Some JWT libraries emit padded segments.
		payload, err = base64.URLEncoding.DecodeString(parts[1])
		if err != nil {
			return time.Time{}, ""
		}
	}
	var claims struct {
		Exp       int64  `json:"exp"`
		UserEmail string `json:"user_email"`
	}
	err = json.Unmarshal(payload, &claims)
	if err != nil {
		return time.Time{}, ""
	}
	if claims.Exp == 0 {
		return time.Time{}, claims.UserEmail
	}
	return time.Unix(claims.Exp, 0).UTC(), claims.UserEmail
}

// newSessionID returns a fresh RFC-4122 v4 UUID string. The Whizlabs
// analytics endpoint records it but does not validate it, so a
// cryptographic-strength UUID is purely "good hygiene".
func newSessionID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// If crypto/rand fails we have much bigger problems. Fall
		// back to a non-cryptographic but still unique string so
		// downstream logging doesn't panic.
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40 //nolint:mnd // RFC 4122 version 4
	b[8] = (b[8] & 0x3f) | 0x80 //nolint:mnd // RFC 4122 variant bits
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(b[0:4]),
		hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]),
		hex.EncodeToString(b[8:10]),
		hex.EncodeToString(b[10:16]),
	)
}
