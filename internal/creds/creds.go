package creds

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type OAuth struct {
	AccessToken           string   `json:"accessToken"`
	RefreshToken          string   `json:"refreshToken"`
	ExpiresAt             int64    `json:"expiresAt"`
	RefreshTokenExpiresAt int64    `json:"refreshTokenExpiresAt"`
	Scopes                []string `json:"scopes"`
	SubscriptionType      string   `json:"subscriptionType"`
	RateLimitTier         string   `json:"rateLimitTier"`
}

type Blob struct {
	ClaudeAiOauth OAuth `json:"claudeAiOauth"`
}

func (b Blob) Validate() error {
	o := b.ClaudeAiOauth
	if !strings.HasPrefix(o.AccessToken, "sk-ant-oat01-") {
		return fmt.Errorf("credentials: missing or malformed accessToken")
	}
	if !strings.HasPrefix(o.RefreshToken, "sk-ant-ort01-") {
		return fmt.Errorf("credentials: missing or malformed refreshToken")
	}
	return nil
}

func ParseBlob(data []byte) (Blob, error) {
	var b Blob
	if err := json.Unmarshal(data, &b); err != nil {
		return Blob{}, fmt.Errorf("credentials: parse: %w", err)
	}
	if err := b.Validate(); err != nil {
		return Blob{}, err
	}
	return b, nil
}

func (o OAuth) Expiry() time.Time { return time.UnixMilli(o.ExpiresAt) }
func (o OAuth) RefreshExpiry() time.Time {
	return time.UnixMilli(o.RefreshTokenExpiresAt)
}

// Path returns the on-disk credentials file location, honoring CLAUDE_CONFIG_DIR.
func Path() string {
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, ".credentials.json")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".claude", ".credentials.json")
}

func ReadFile() (Blob, error) {
	data, err := os.ReadFile(Path())
	if err != nil {
		return Blob{}, fmt.Errorf("credentials: read %s: %w", Path(), err)
	}
	return ParseBlob(data)
}

func WriteFile(b Blob) error {
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return err
	}
	p := Path()
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	return os.WriteFile(p, append(data, '\n'), 0o600)
}

func FileExists() bool {
	_, err := os.Stat(Path())
	return err == nil
}
