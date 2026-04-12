package session_test

import (
	"testing"
	"time"

	"github.com/meigma/whzbox/internal/core/session"
)

var baseTime = time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC) //nolint:gochecknoglobals // test fixture

func TestTokens_AccessValid(t *testing.T) {
	tests := []struct {
		name  string
		token session.Tokens
		now   time.Time
		want  bool
	}{
		{
			name:  "valid with headroom",
			token: session.Tokens{AccessToken: "x", AccessTokenExpiresAt: baseTime.Add(time.Hour)},
			now:   baseTime,
			want:  true,
		},
		{
			name:  "already expired",
			token: session.Tokens{AccessToken: "x", AccessTokenExpiresAt: baseTime.Add(-time.Minute)},
			now:   baseTime,
			want:  false,
		},
		{
			name:  "expiry equals now counts as expired",
			token: session.Tokens{AccessToken: "x", AccessTokenExpiresAt: baseTime},
			now:   baseTime,
			want:  false,
		},
		{
			name:  "empty access token",
			token: session.Tokens{AccessTokenExpiresAt: baseTime.Add(time.Hour)},
			now:   baseTime,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.token.AccessValid(tt.now)
			if got != tt.want {
				t.Errorf("AccessValid(%v) = %v, want %v", tt.now, got, tt.want)
			}
		})
	}
}

func TestTokens_AccessNearExpiry(t *testing.T) {
	window := 10 * time.Minute

	tests := []struct {
		name  string
		token session.Tokens
		want  bool
	}{
		{
			name:  "far from expiry",
			token: session.Tokens{AccessToken: "x", AccessTokenExpiresAt: baseTime.Add(time.Hour)},
			want:  false,
		},
		{
			name:  "just outside window",
			token: session.Tokens{AccessToken: "x", AccessTokenExpiresAt: baseTime.Add(11 * time.Minute)},
			want:  false,
		},
		{
			name:  "exactly at window edge",
			token: session.Tokens{AccessToken: "x", AccessTokenExpiresAt: baseTime.Add(10 * time.Minute)},
			want:  true,
		},
		{
			name:  "inside window",
			token: session.Tokens{AccessToken: "x", AccessTokenExpiresAt: baseTime.Add(5 * time.Minute)},
			want:  true,
		},
		{
			name:  "already expired",
			token: session.Tokens{AccessToken: "x", AccessTokenExpiresAt: baseTime.Add(-time.Minute)},
			want:  true,
		},
		{
			name:  "empty access token always near expiry",
			token: session.Tokens{AccessTokenExpiresAt: baseTime.Add(time.Hour)},
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.token.AccessNearExpiry(baseTime, window)
			if got != tt.want {
				t.Errorf("AccessNearExpiry(%v, %v) = %v, want %v", baseTime, window, got, tt.want)
			}
		})
	}
}

func TestTokens_Refreshable(t *testing.T) {
	tests := []struct {
		name  string
		token session.Tokens
		want  bool
	}{
		{
			name: "valid refresh token",
			token: session.Tokens{
				RefreshToken:          "r",
				RefreshTokenExpiresAt: baseTime.Add(24 * time.Hour),
			},
			want: true,
		},
		{
			name: "expired refresh token",
			token: session.Tokens{
				RefreshToken:          "r",
				RefreshTokenExpiresAt: baseTime.Add(-time.Hour),
			},
			want: false,
		},
		{
			name: "empty refresh token",
			token: session.Tokens{
				RefreshTokenExpiresAt: baseTime.Add(24 * time.Hour),
			},
			want: false,
		},
		{
			name: "expiry equals now",
			token: session.Tokens{
				RefreshToken:          "r",
				RefreshTokenExpiresAt: baseTime,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.token.Refreshable(baseTime)
			if got != tt.want {
				t.Errorf("Refreshable(%v) = %v, want %v", baseTime, got, tt.want)
			}
		})
	}
}
