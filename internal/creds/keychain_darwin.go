//go:build darwin

package creds

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const keychainService = "Claude Code-credentials"

func keychainAccount() string {
	if u := os.Getenv("USER"); u != "" {
		return u
	}
	return ""
}

func ReadKeychain() (Blob, error) {
	out, err := exec.Command("security", "find-generic-password", "-s", keychainService, "-w").Output()
	if err != nil {
		return Blob{}, fmt.Errorf("keychain: %w (is Claude Code logged in?)", err)
	}
	return ParseBlob([]byte(strings.TrimSpace(string(out))))
}

func WriteKeychain(b Blob) error {
	data, err := json.Marshal(b)
	if err != nil {
		return err
	}
	args := []string{"add-generic-password", "-U", "-s", keychainService}
	if acct := keychainAccount(); acct != "" {
		args = append(args, "-a", acct)
	}
	args = append(args, "-w", string(data))
	if out, err := exec.Command("security", args...).CombinedOutput(); err != nil {
		return fmt.Errorf("keychain write: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
