package meta

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Account struct {
	AccountUuid      string          `json:"accountUuid"`
	EmailAddress     string          `json:"emailAddress"`
	OrganizationName string          `json:"organizationName"`
	SeatTier         string          `json:"seatTier"`
	OrganizationRole string          `json:"organizationRole"`
	Raw              json.RawMessage `json:"raw,omitempty"`
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
// Raw keeps the untouched oauthAccount object so it can be written back later.
func ReadAccount() (Account, error) {
	data, err := os.ReadFile(claudeJSON())
	if err != nil {
		return Account{}, fmt.Errorf("meta: read %s: %w", claudeJSON(), err)
	}
	var doc struct {
		OauthAccount json.RawMessage `json:"oauthAccount"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return Account{}, fmt.Errorf("meta: parse %s: %w", claudeJSON(), err)
	}
	var acct Account
	if len(doc.OauthAccount) > 0 {
		if err := json.Unmarshal(doc.OauthAccount, &acct); err != nil {
			return Account{}, fmt.Errorf("meta: parse oauthAccount: %w", err)
		}
		acct.Raw = json.RawMessage(doc.OauthAccount)
	}
	return acct, nil
}

// WriteAccount replaces the oauthAccount block in ~/.claude.json with raw,
// preserving every other top-level key. A backup is left at .claude.json.ccauth-bak.
func WriteAccount(raw json.RawMessage) error {
	p := claudeJSON()
	data, err := os.ReadFile(p)
	if err != nil {
		return fmt.Errorf("meta: read %s: %w", p, err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("meta: parse %s: %w", p, err)
	}
	_ = os.WriteFile(p+".ccauth-bak", data, 0o600)
	doc["oauthAccount"] = raw
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("meta: serialize %s: %w", p, err)
	}
	mode := os.FileMode(0o600)
	if info, serr := os.Stat(p); serr == nil {
		mode = info.Mode().Perm()
	}
	return os.WriteFile(p, append(out, '\n'), mode)
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
