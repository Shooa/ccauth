package meta

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func sampleClaudeJSON() []byte {
	return []byte(`{
  "numStartups": 42,
  "oauthAccount": {
    "accountUuid": "uuid-old",
    "emailAddress": "old@example.com",
    "organizationName": "OldOrg",
    "billingType": "subscription",
    "hasExtraUsageEnabled": true
  },
  "projects": {"/tmp/x": {"allowedTools": []}}
}`)
}

func withTempConfig(t *testing.T, content []byte) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	if content != nil {
		if err := os.WriteFile(filepath.Join(dir, ".claude.json"), content, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestReadAccountRaw(t *testing.T) {
	withTempConfig(t, sampleClaudeJSON())
	acct, err := ReadAccount()
	if err != nil {
		t.Fatal(err)
	}
	if acct.AccountUuid != "uuid-old" || acct.EmailAddress != "old@example.com" {
		t.Errorf("fields: %+v", acct)
	}
	if len(acct.Raw) == 0 {
		t.Fatal("Raw not captured")
	}
	var raw map[string]any
	json.Unmarshal(acct.Raw, &raw)
	if raw["billingType"] != "subscription" {
		t.Errorf("Raw lost fields: %s", acct.Raw)
	}
}

func TestWriteAccountPreservesOtherKeys(t *testing.T) {
	withTempConfig(t, sampleClaudeJSON())
	newAcct := []byte(`{"accountUuid":"uuid-new","emailAddress":"new@example.com"}`)
	if err := WriteAccount(newAcct); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(os.Getenv("CLAUDE_CONFIG_DIR"), ".claude.json"))
	var doc struct {
		NumStartups  int             `json:"numStartups"`
		Projects     json.RawMessage `json:"projects"`
		OauthAccount struct {
			AccountUuid         string `json:"accountUuid"`
			BillingType         string `json:"billingType"`
			HasExtraUsageEnable bool   `json:"hasExtraUsageEnabled"`
		} `json:"oauthAccount"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("result not valid JSON: %v\n%s", err, data)
	}
	if doc.NumStartups != 42 {
		t.Errorf("numStartups lost: %d", doc.NumStartups)
	}
	if len(doc.Projects) == 0 {
		t.Error("projects key lost")
	}
	if doc.OauthAccount.AccountUuid != "uuid-new" {
		t.Errorf("oauthAccount not replaced: %s", doc.OauthAccount.AccountUuid)
	}
	if doc.OauthAccount.BillingType != "" {
		t.Error("old oauthAccount fields should not leak into new account")
	}

	// backup exists and holds the old content
	bak, err := os.ReadFile(filepath.Join(os.Getenv("CLAUDE_CONFIG_DIR"), ".claude.json.ccauth-bak"))
	if err != nil {
		t.Fatal("backup missing")
	}
	if !json.Valid(bak) {
		t.Error("backup not valid JSON")
	}
}

func TestWriteAccountMissingFile(t *testing.T) {
	withTempConfig(t, nil)
	if err := WriteAccount([]byte(`{}`)); err == nil {
		t.Error("expected error when .claude.json missing")
	}
}

func TestWriteAccountPermPreserved(t *testing.T) {
	dir := withTempConfig(t, sampleClaudeJSON())
	p := filepath.Join(dir, ".claude.json")
	os.Chmod(p, 0o644)
	if err := WriteAccount([]byte(`{"accountUuid":"u"}`)); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(p)
	if perm := info.Mode().Perm(); perm != 0o644 {
		t.Errorf("perm = %o, want 644 (preserved)", perm)
	}
}
