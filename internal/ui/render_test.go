package ui_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/meigma/whzbox/internal/core/sandbox"
	"github.com/meigma/whzbox/internal/core/session"
	"github.com/meigma/whzbox/internal/ui"
)

func sampleSandbox() *sandbox.Sandbox {
	return &sandbox.Sandbox{
		Kind: sandbox.KindAWS,
		Slug: "aws-sandbox",
		Credentials: sandbox.Credentials{
			AccessKey: "AKIA0123456789ABCDEF",
			SecretKey: "shhh",
		},
		Console: sandbox.Console{
			URL:      "https://111111111111.signin.aws.amazon.com/console?region=us-east-1",
			Username: "Whiz_User_1.2",
			Password: "pw-uuid",
		},
		Identity: sandbox.Identity{
			Account: "111111111111",
			UserID:  "AIDASAMPLE",
			ARN:     "arn:aws:iam::111111111111:user/Whiz_User_1.2",
			Region:  "us-east-1",
		},
		StartedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}
}

func TestRenderSandbox_IncludesAllFields(t *testing.T) {
	var buf bytes.Buffer
	ui.RenderSandbox(&buf, sampleSandbox())
	out := buf.String()

	must := []string{
		"Account",
		"111111111111",
		"ARN",
		"arn:aws:iam::111111111111:user/Whiz_User_1.2",
		"Region",
		"us-east-1",
		"Console",
		"https://111111111111.signin.aws.amazon.com/console",
		"Whiz_User_1.2",
		"pw-uuid",
		"AWS_ACCESS_KEY_ID",
		"AKIA0123456789ABCDEF",
		"AWS_SECRET_ACCESS_KEY",
		"shhh",
		"Destroy with:",
		"whzbox destroy",
	}
	for _, m := range must {
		if !strings.Contains(out, m) {
			t.Errorf("output missing %q\n%s", m, out)
		}
	}
}

func TestRenderSandbox_NilIsNoop(t *testing.T) {
	var buf bytes.Buffer
	ui.RenderSandbox(&buf, nil)
	if buf.Len() != 0 {
		t.Errorf("nil sandbox should produce no output, got: %q", buf.String())
	}
}

func TestRenderSandbox_GCPConsoleOnly(t *testing.T) {
	sb := &sandbox.Sandbox{
		Kind: sandbox.KindGCP,
		Identity: sandbox.Identity{
			ProjectID:   "project-12345",
			ProjectName: "project-12345",
		},
		Console: sandbox.Console{
			URL:      "https://console.cloud.google.com/",
			Username: "student@example.com",
			Password: "gcp-password",
		},
		ExpiresAt: time.Now().Add(time.Hour),
	}

	var buf bytes.Buffer
	ui.RenderSandbox(&buf, sb)
	out := buf.String()
	for _, want := range []string{"Project ID", "project-12345", "Browser console only", "student@example.com", "gcp-password"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n%s", want, out)
		}
	}
	if strings.Contains(out, "AWS_ACCESS_KEY_ID") {
		t.Errorf("GCP output should not include AWS credentials:\n%s", out)
	}
}

func TestRenderSandboxJSON_GCPProjectIdentity(t *testing.T) {
	sb := &sandbox.Sandbox{
		Kind:     sandbox.KindGCP,
		Slug:     "gcp-sandbox",
		Identity: sandbox.Identity{ProjectID: "project-12345", ProjectName: "project-12345"},
		Console: sandbox.Console{
			URL:      "https://console.cloud.google.com/",
			Username: "student@example.com",
			Password: "secret",
		},
		StartedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}

	var buf bytes.Buffer
	if err := ui.RenderSandboxJSON(&buf, sb); err != nil {
		t.Fatalf("RenderSandboxJSON: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	identity, _ := got["identity"].(map[string]any)
	if identity["project_id"] != "project-12345" {
		t.Errorf("project_id: got %v", identity["project_id"])
	}
	credentials, _ := got["credentials"].(map[string]any)
	if credentials["access_key"] != "" || credentials["secret_key"] != "" {
		t.Errorf("GCP programmatic credentials should be empty: %v", credentials)
	}
}

func TestRenderStatus_NotLoggedIn(t *testing.T) {
	var buf bytes.Buffer
	ui.RenderStatus(&buf, session.Tokens{}, false)
	out := buf.String()

	if !strings.Contains(out, "(not logged in)") {
		t.Errorf("missing not-logged-in marker:\n%s", out)
	}
	if strings.Contains(out, "Active sandbox") {
		t.Errorf("status should not render sandbox state:\n%s", out)
	}
}

func TestRenderStatus_SessionOnly(t *testing.T) {
	var buf bytes.Buffer
	tokens := session.Tokens{
		UserEmail:             "alice@example.com",
		AccessToken:           "a",
		RefreshToken:          "r",
		AccessTokenExpiresAt:  time.Now().Add(12 * time.Hour),
		RefreshTokenExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	ui.RenderStatus(&buf, tokens, true)
	out := buf.String()

	if !strings.Contains(out, "alice@example.com") {
		t.Errorf("missing email: %s", out)
	}
	if !strings.Contains(out, "Refresh") {
		t.Errorf("missing refresh line: %s", out)
	}
	if strings.Contains(out, "Active sandbox") {
		t.Errorf("status should not render sandbox state: %s", out)
	}
}

func TestRenderSandbox_ExpiredTimestamp(t *testing.T) {
	sb := sampleSandbox()
	sb.ExpiresAt = time.Now().Add(-time.Hour)

	var buf bytes.Buffer
	ui.RenderSandbox(&buf, sb)
	if !strings.Contains(buf.String(), "expired") {
		t.Errorf("expired timestamp should be labelled: %s", buf.String())
	}
}
