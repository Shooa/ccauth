package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/Shooa/ccauth/internal/creds"
	"github.com/Shooa/ccauth/internal/meta"
)

func testBlob() creds.Blob {
	return creds.Blob{ClaudeAiOauth: creds.OAuth{
		AccessToken:           "sk-ant-oat01-xxx",
		RefreshToken:          "sk-ant-ort01-yyy",
		ExpiresAt:             1790946617795,
		RefreshTokenExpiresAt: 1790946617795,
		Scopes:                []string{"user:inference"},
		SubscriptionType:      "max",
	}}
}

func withTempDir(t *testing.T) {
	t.Helper()
	t.Setenv("CCAUTH_DIR", t.TempDir())
}

func TestSaveLoadListDelete(t *testing.T) {
	withTempDir(t)

	if err := Save("work", testBlob(), meta.Account{EmailAddress: "a@b.c", AccountUuid: "u1"}, Settings{}); err != nil {
		t.Fatal(err)
	}
	if err := Save("personal", testBlob(), meta.Account{EmailAddress: "p@q.r", AccountUuid: "u2"}, Settings{}); err != nil {
		t.Fatal(err)
	}

	p, err := Load("work")
	if err != nil {
		t.Fatal(err)
	}
	if p.Credentials.ClaudeAiOauth.AccessToken != "sk-ant-oat01-xxx" {
		t.Errorf("roundtrip failed: %+v", p.Credentials)
	}
	if p.Account.EmailAddress != "a@b.c" {
		t.Errorf("meta lost: %+v", p.Account)
	}

	list, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].Name != "personal" || list[1].Name != "work" {
		t.Errorf("list: %+v", list)
	}

	if err := Delete("work"); err != nil {
		t.Fatal(err)
	}
	list, _ = List()
	if len(list) != 1 {
		t.Errorf("delete failed: %+v", list)
	}
}

func TestFilePermissions(t *testing.T) {
	withTempDir(t)
	if err := Save("sec", testBlob(), meta.Account{}, Settings{}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(os.Getenv("CCAUTH_DIR"), "sec.json"))
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("profile file perm = %o, want 600", perm)
	}
}

func TestValidName(t *testing.T) {
	for _, ok := range []string{"work", "personal-1", "a.b_c", "A9"} {
		if ValidName(ok) != nil {
			t.Errorf("%q should be valid", ok)
		}
	}
	for _, bad := range []string{"", "../etc", "a/b", ".hidden", "-x", "with space"} {
		if ValidName(bad) == nil {
			t.Errorf("%q should be invalid", bad)
		}
	}
	if err := Save("../evil", testBlob(), meta.Account{}, Settings{}); err == nil {
		t.Error("Save must reject path traversal")
	}
}

func TestSettingsRoundtrip(t *testing.T) {
	withTempDir(t)
	data := json.RawMessage(`{"permissions":{"allow":["Bash"]}}`)
	if err := Save("s", testBlob(), meta.Account{}, Settings{Included: true, Data: data}); err != nil {
		t.Fatal(err)
	}
	p, err := Load("s")
	if err != nil {
		t.Fatal(err)
	}
	if !p.Settings.Included {
		t.Fatal("settings flag lost")
	}
	var got, want map[string]any
	if json.Unmarshal(p.Settings.Data, &got) != nil {
		t.Fatalf("settings data invalid JSON: %s", p.Settings.Data)
	}
	json.Unmarshal(data, &want)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("settings lost: %s", p.Settings.Data)
	}
}

func TestLoadMissing(t *testing.T) {
	withTempDir(t)
	if _, err := Load("nope"); err == nil {
		t.Error("expected error for missing profile")
	}
}

func TestSavedAtPopulated(t *testing.T) {
	withTempDir(t)
	before := time.Now().Add(-time.Second)
	if err := Save("t", testBlob(), meta.Account{}, Settings{}); err != nil {
		t.Fatal(err)
	}
	p, _ := Load("t")
	if p.SavedAt.Before(before) || p.SavedAt.After(time.Now().Add(time.Second)) {
		t.Errorf("SavedAt not populated: %v", p.SavedAt)
	}
}
