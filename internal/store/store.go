package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	"github.com/Shooa/ccauth/internal/creds"
	"github.com/Shooa/ccauth/internal/meta"
)

const version = 1

type Settings struct {
	Included bool            `json:"included"`
	Data     json.RawMessage `json:"data,omitempty"`
}

type Profile struct {
	Version     int          `json:"version"`
	Name        string       `json:"name"`
	SavedAt     time.Time    `json:"savedAt"`
	Credentials creds.Blob   `json:"credentials"`
	Account     meta.Account `json:"account"`
	Settings    Settings     `json:"settings"`
}

var nameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,63}$`)

func ValidName(name string) error {
	if !nameRe.MatchString(name) {
		return fmt.Errorf("invalid profile name %q (allowed: letters, digits, '.', '_', '-')", name)
	}
	return nil
}

// Dir returns the profile store root, honoring CCAUTH_DIR (mostly for tests).
func Dir() string {
	if d := os.Getenv("CCAUTH_DIR"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".ccauth", "profiles")
}

func profilePath(name string) string {
	return filepath.Join(Dir(), name+".json")
}

func Save(name string, b creds.Blob, acct meta.Account, settings Settings) error {
	if err := ValidName(name); err != nil {
		return err
	}
	p := Profile{
		Version:     version,
		Name:        name,
		SavedAt:     time.Now(),
		Credentials: b,
		Account:     acct,
		Settings:    settings,
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(Dir(), 0o700); err != nil {
		return err
	}
	return os.WriteFile(profilePath(name), append(data, '\n'), 0o600)
}

func Load(name string) (Profile, error) {
	if err := ValidName(name); err != nil {
		return Profile{}, err
	}
	data, err := os.ReadFile(profilePath(name))
	if err != nil {
		return Profile{}, fmt.Errorf("profile %q: %w", name, err)
	}
	var p Profile
	if err := json.Unmarshal(data, &p); err != nil {
		return Profile{}, fmt.Errorf("profile %q: parse: %w", name, err)
	}
	return p, nil
}

func List() ([]Profile, error) {
	entries, err := os.ReadDir(Dir())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Profile
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(Dir(), e.Name()))
		if err != nil {
			continue
		}
		var p Profile
		if json.Unmarshal(data, &p) != nil {
			continue
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func Delete(name string) error {
	if err := ValidName(name); err != nil {
		return err
	}
	if _, err := os.Stat(profilePath(name)); err != nil {
		return fmt.Errorf("profile %q not found", name)
	}
	return os.Remove(profilePath(name))
}
