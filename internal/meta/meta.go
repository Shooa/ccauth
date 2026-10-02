package meta

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Account struct {
	AccountUuid      string `json:"accountUuid"`
	EmailAddress     string `json:"emailAddress"`
	OrganizationName string `json:"organizationName"`
	SeatTier         string `json:"seatTier"`
	OrganizationRole string `json:"organizationRole"`
}

// ConfigDir returns Claude Code's config dir, honoring CLAUDE_CONFIG_DIR.
func ConfigDir() string {
	if d := os.Getenv("CLAUDE_CONFIG_DIR"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".claude")
}

func claudeJSON() string {
	if d := os.Getenv("CLAUDE_CONFIG_DIR"); d != "" {
		return filepath.Join(d, ".claude.json")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".claude.json")
}

func SettingsPath() string {
	return filepath.Join(ConfigDir(), "settings.json")
}

// ReadAccount extracts the oauthAccount block from ~/.claude.json.
func ReadAccount() (Account, error) {
	data, err := os.ReadFile(claudeJSON())
	if err != nil {
		return Account{}, fmt.Errorf("meta: read %s: %w", claudeJSON(), err)
	}
	var doc struct {
		OauthAccount Account `json:"oauthAccount"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return Account{}, fmt.Errorf("meta: parse %s: %w", claudeJSON(), err)
	}
	return doc.OauthAccount, nil
}

// ReadSettings returns raw settings.json content.
func ReadSettings() ([]byte, error) {
	return os.ReadFile(SettingsPath())
}

func WriteSettings(data []byte) error {
	p := SettingsPath()
	if cur, err := os.ReadFile(p); err == nil {
		_ = os.WriteFile(p+".ccauth-bak", cur, 0o600)
	}
	return os.WriteFile(p, data, 0o600)
}
