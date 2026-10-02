package creds

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validBlob = `{"claudeAiOauth":{"accessToken":"sk-ant-oat01-abc","refreshToken":"sk-ant-ort01-def","expiresAt":1790946617795,"refreshTokenExpiresAt":1790946617795,"scopes":["user:inference"],"subscriptionType":"max","rateLimitTier":"p1"}}`

func TestParseBlob(t *testing.T) {
	b, err := ParseBlob([]byte(validBlob))
	if err != nil {
		t.Fatal(err)
	}
	if b.ClaudeAiOauth.SubscriptionType != "max" {
		t.Errorf("unexpected: %+v", b)
	}
}

func TestParseBlobRejectsGarbage(t *testing.T) {
	if _, err := ParseBlob([]byte(`{}`)); err == nil {
		t.Error("empty oauth must fail validation")
	}
	if _, err := ParseBlob([]byte(`not json`)); err == nil {
		t.Error("garbage must fail")
	}
}

func TestFilePathRespectsConfigDir(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", "/tmp/custom-cc")
	if got := Path(); got != filepath.Join("/tmp/custom-cc", ".credentials.json") {
		t.Errorf("Path() = %q", got)
	}
}

func TestFileWriteRead(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	want, _ := ParseBlob([]byte(validBlob))
	if err := WriteFile(want); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, ".credentials.json"))
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("credentials file perm = %o, want 600", perm)
	}
	got, err := ReadFile()
	if err != nil {
		t.Fatal(err)
	}
	if got.ClaudeAiOauth.AccessToken != want.ClaudeAiOauth.AccessToken {
		t.Error("roundtrip mismatch")
	}
	if !FileExists() {
		t.Error("FileExists should be true")
	}
}

func TestValidatePrefixes(t *testing.T) {
	b := Blob{ClaudeAiOauth: OAuth{AccessToken: "wrong", RefreshToken: "sk-ant-ort01-x"}}
	if err := b.Validate(); err == nil || !strings.Contains(err.Error(), "accessToken") {
		t.Errorf("want accessToken error, got %v", err)
	}
}
