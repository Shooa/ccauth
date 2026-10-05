package store

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/Shooa/ccauth/internal/meta"
)

// TestUpdateCredentialsFollowsRotation: a snapshot's credentials can be
// replaced in place while name/account/settings survive; UpdatedAt moves.
func TestUpdateCredentialsFollowsRotation(t *testing.T) {
	withTempDir(t)

	old := testBlob()
	if err := Save("work", old, meta.Account{EmailAddress: "a@b.c", AccountUuid: "u1"}, Settings{Included: true, Data: []byte(`{"x":1}`)}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond) // ensure UpdatedAt > SavedAt

	rotated := testBlob()
	rotated.ClaudeAiOauth.AccessToken = "sk-ant-oat01-rotated"
	rotated.ClaudeAiOauth.ExpiresAt = time.Now().Add(8 * time.Hour).UnixMilli()

	if err := UpdateCredentials("work", rotated); err != nil {
		t.Fatal(err)
	}
	p, err := Load("work")
	if err != nil {
		t.Fatal(err)
	}
	if p.Credentials.ClaudeAiOauth.AccessToken != "sk-ant-oat01-rotated" {
		t.Fatal("credentials not updated")
	}
	if p.UpdatedAt.IsZero() {
		t.Fatal("UpdatedAt not set")
	}
	if p.Account.EmailAddress != "a@b.c" || p.Account.AccountUuid != "u1" {
		t.Fatal("account metadata lost")
	}
	if !p.Settings.Included {
		t.Fatal("settings lost (Included)")
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, p.Settings.Data); err != nil {
		t.Fatal(err)
	}
	if compact.String() != `{"x":1}` {
		t.Fatalf("settings data changed: %s", compact.String())
	}
	if p.CredentialsDiffer(rotated) {
		t.Fatal("CredentialsDiffer true for identical blob")
	}
	if !p.CredentialsDiffer(old) {
		t.Fatal("CredentialsDiffer false for different blob")
	}
}

func TestUpdateCredentialsUnknownProfile(t *testing.T) {
	withTempDir(t)
	if err := UpdateCredentials("nope", testBlob()); err == nil {
		t.Fatal("expected error for missing profile")
	}
}

// TestCredentialsDifferOrderStable: JSON comparison must not report a
// difference when only struct construction order differs — both sides go
// through the same marshal path, so field order is stable.
func TestCredentialsDifferOrderStable(t *testing.T) {
	withTempDir(t)
	if err := Save("p", testBlob(), meta.Account{}, Settings{}); err != nil {
		t.Fatal(err)
	}
	p, err := Load("p")
	if err != nil {
		t.Fatal(err)
	}
	if p.CredentialsDiffer(testBlob()) {
		t.Fatal("identical credentials reported as different")
	}
	_ = filepath.Join // keep filepath import if unused later
}
