package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Shooa/ccauth/internal/creds"
	"github.com/Shooa/ccauth/internal/meta"
	"github.com/Shooa/ccauth/internal/selfupdate"
	"github.com/Shooa/ccauth/internal/store"
	"github.com/Shooa/ccauth/internal/ui"
)

var version = "dev"

func usage(w *os.File) {
	fmt.Fprint(w, `ccauth — save/restore Claude Code authorization profiles

Usage:
  ccauth save <name> [--settings]   Save current auth (and optionally settings.json)
  ccauth restore [name] [--settings] Restore profile; without name: interactive picker
  ccauth list                       List profiles with token expiry
  ccauth show <name>                Profile details
  ccauth current                    Show live credentials info
  ccauth remove <name>              Delete profile
  ccauth update                     Update ccauth to the latest release

Shortcuts: s=save r=restore/use cur=current up=update rm=remove ls=list

Storage: ~/.ccauth/profiles (override with CCAUTH_DIR)
Credentials source: macOS Keychain / ~/.claude/.credentials.json
  (honors CLAUDE_CONFIG_DIR)
`)
}

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "ccauth: "+format+"\n", args...)
	os.Exit(1)
}

func main() {
	if offerUpdate(context.Background(), os.Args[1:]) {
		return
	}
	if len(os.Args) < 2 {
		usage(os.Stderr)
		os.Exit(2)
	}
	cmd, args := os.Args[1], os.Args[2:]
	var err error
	switch cmd {
	case "save", "s":
		err = cmdSave(args)
	case "restore", "use", "r":
		err = cmdRestore(args)
	case "list", "ls":
		err = cmdList()
	case "show":
		err = cmdShow(args)
	case "current", "cur":
		err = cmdCurrent()
	case "remove", "rm":
		err = cmdRemove(args)
	case "update", "up":
		err = cmdUpdate()
	case "version", "--version", "-v":
		fmt.Println("ccauth " + version)
	case "help", "--help", "-h":
		usage(os.Stdout)
	default:
		fmt.Fprintf(os.Stderr, "ccauth: unknown command %q\n\n", cmd)
		usage(os.Stderr)
		os.Exit(2)
	}
	if err != nil {
		die("%v", err)
	}
}

func boolFlag(fs *flag.FlagSet, name, help string) *bool {
	fs.Usage = func() {}
	return fs.Bool(name, false, help)
}

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	fs.Usage = func() { usage(os.Stderr) }
	return fs
}

func cmdSave(args []string) error {
	fs := newFlagSet("save")
	withSettings := boolFlag(fs, "settings", "also save ~/.claude/settings.json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: ccauth save <name> [--settings]")
	}
	name := fs.Arg(0)

	blob, src, err := creds.ReadCurrent()
	if err != nil {
		return err
	}
	acct, err := meta.ReadAccount()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ccauth: warn: %v (saving without account metadata)\n", err)
	}
	settings := store.Settings{Included: false}
	if *withSettings {
		data, serr := meta.ReadSettings()
		if serr != nil {
			return fmt.Errorf("read settings: %w", serr)
		}
		if !json.Valid(data) {
			return fmt.Errorf("settings.json is not valid JSON")
		}
		settings = store.Settings{Included: true, Data: json.RawMessage(data)}
	}
	if err := store.Save(name, blob, acct, settings); err != nil {
		return err
	}
	fmt.Printf("Saved profile %q from %s, refresh expires %s\n", name, src, ui.FmtExpiry(blob.ClaudeAiOauth.RefreshTokenExpiresAt, time.Now()))
	return nil
}

func cmdRestore(args []string) error {
	fs := newFlagSet("restore")
	withSettings := boolFlag(fs, "settings", "also restore saved settings.json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 1 {
		return fmt.Errorf("usage: ccauth restore [name] [--settings]")
	}
	if fs.NArg() == 0 {
		// Interactive: show numbered list, pick with a number.
		profiles, err := store.List()
		if err != nil {
			return err
		}
		if len(profiles) == 0 {
			fmt.Println("No profiles. Create one: ccauth save <name>")
			return nil
		}
		active := activeProfileName()
		now := time.Now()
		fmt.Println("Available profiles:")
		for i, p := range profiles {
			mark := " "
			if p.Name == active {
				mark = "*"
			}
			fmt.Printf("  %2d) %s %-12s %-28s %s\n", i+1, mark, p.Name,
				ui.Truncate(p.Account.EmailAddress, 28),
				expiryCell(p.Credentials.ClaudeAiOauth.RefreshTokenExpiresAt, now))
		}
		fmt.Fprint(os.Stderr, "Number to activate (empty = cancel): ")
		answer, readErr := bufio.NewReader(os.Stdin).ReadString('\n')
		if readErr != nil && len(answer) == 0 {
			return fmt.Errorf("cancelled")
		}
		answer = strings.TrimSpace(answer)
		if answer == "" {
			fmt.Println("Cancelled.")
			return nil
		}
		n, aerr := strconv.Atoi(answer)
		if aerr != nil || n < 1 || n > len(profiles) {
			return fmt.Errorf("invalid number %q (1-%d)", answer, len(profiles))
		}
		return doRestore(profiles[n-1], *withSettings)
	}
	name := fs.Arg(0)
	p, err := store.Load(name)
	if err != nil {
		return err
	}
	return doRestore(p, *withSettings)
}

func doRestore(p store.Profile, withSettings bool) error {
	name := p.Name
	if err := p.Credentials.Validate(); err != nil {
		return fmt.Errorf("profile %q: %v", name, err)
	}
	if err := creds.Restore(p.Credentials); err != nil {
		return err
	}
	fmt.Printf("Restored profile %q (%s)\n", name, p.Account.EmailAddress)
	if withSettings {
		if !p.Settings.Included {
			return fmt.Errorf("profile %q has no saved settings", name)
		}
		if err := meta.WriteSettings(p.Settings.Data); err != nil {
			return fmt.Errorf("write settings: %w", err)
		}
		fmt.Println("Restored settings.json (previous copy: settings.json.ccauth-bak)")
	}
	fmt.Println("Restart running Claude Code sessions to pick up the new credentials.")
	return nil
}

func activeProfileName() string {
	acct, err := meta.ReadAccount()
	if err != nil || acct.AccountUuid == "" {
		return ""
	}
	profiles, err := store.List()
	if err != nil {
		return ""
	}
	for _, p := range profiles {
		if p.Account.AccountUuid != "" && p.Account.AccountUuid == acct.AccountUuid {
			return p.Name
		}
	}
	return ""
}

func expiryCell(ms int64, now time.Time) string {
	return ui.FmtExpiry(ms, now)
}

func cmdList() error {
	if len(os.Args) > 2 {
		return fmt.Errorf("usage: ccauth list")
	}
	profiles, err := store.List()
	if err != nil {
		return err
	}
	if len(profiles) == 0 {
		fmt.Println("No profiles. Create one: ccauth save <name>")
		return nil
	}
	active := activeProfileName()
	now := time.Now()
	headers := []string{"", "NAME", "EMAIL/ORG", "SUBSCRIPTION", "ACCESS EXPIRES", "REFRESH EXPIRES", "SETTINGS"}
	var rows [][]string
	for _, p := range profiles {
		mark := ""
		if p.Name == active {
			mark = "*"
		}
		email := p.Account.EmailAddress
		if org := p.Account.OrganizationName; org != "" {
			email = email + " / " + org
		}
		set := "-"
		if p.Settings.Included {
			set = "yes"
		}
		rows = append(rows, []string{
			mark,
			p.Name,
			ui.Truncate(email, 46),
			ui.Truncate(p.Credentials.ClaudeAiOauth.SubscriptionType, 14),
			expiryCell(p.Credentials.ClaudeAiOauth.ExpiresAt, now),
			expiryCell(p.Credentials.ClaudeAiOauth.RefreshTokenExpiresAt, now),
			set,
		})
	}
	fmt.Print(ui.Table(headers, rows))
	return nil
}

func cmdShow(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: ccauth show <name>")
	}
	p, err := store.Load(args[0])
	if err != nil {
		return err
	}
	now := time.Now()
	o := p.Credentials.ClaudeAiOauth
	fmt.Printf("Profile:      %s (saved %s)\n", p.Name, p.SavedAt.Local().Format("2006-01-02 15:04"))
	if p.Account.EmailAddress != "" {
		fmt.Printf("Account:      %s (%s)\n", p.Account.EmailAddress, p.Account.SeatTier)
		fmt.Printf("Organization: %s [%s]\n", p.Account.OrganizationName, p.Account.OrganizationRole)
		fmt.Printf("Account UUID: %s\n", p.Account.AccountUuid)
	}
	fmt.Printf("Subscription: %s (%s)\n", o.SubscriptionType, o.RateLimitTier)
	fmt.Printf("Access token: …%s  expires %s\n", last(o.AccessToken, 6), ui.FmtExpiry(o.ExpiresAt, now))
	fmt.Printf("Refresh:      …%s  expires %s\n", last(o.RefreshToken, 6), ui.FmtExpiry(o.RefreshTokenExpiresAt, now))
	fmt.Printf("Scopes:       %s\n", joinScopes(o.Scopes))
	fmt.Printf("Settings:     %s\n", settingsStatus(p))
	if p.Name == activeProfileName() {
		fmt.Println("Status:       active")
	}
	return nil
}

func cmdCurrent() error {
	if len(os.Args) > 2 {
		return fmt.Errorf("usage: ccauth current")
	}
	blob, src, err := creds.ReadCurrent()
	if err != nil {
		return err
	}
	o := blob.ClaudeAiOauth
	now := time.Now()
	fmt.Printf("Source:       %s\n", src)
	acct, merr := meta.ReadAccount()
	if merr == nil && acct.EmailAddress != "" {
		fmt.Printf("Account:      %s (%s / %s)\n", acct.EmailAddress, acct.OrganizationName, acct.SeatTier)
	}
	fmt.Printf("Subscription: %s (%s)\n", o.SubscriptionType, o.RateLimitTier)
	fmt.Printf("Access:       expires %s\n", ui.FmtExpiry(o.ExpiresAt, now))
	fmt.Printf("Refresh:      expires %s\n", ui.FmtExpiry(o.RefreshTokenExpiresAt, now))
	if active := activeProfileName(); active != "" {
		fmt.Printf("Profile:      %s (saved)\n", active)
	}
	return nil
}

func cmdRemove(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: ccauth remove <name>")
	}
	return store.Delete(args[0])
}

func cmdUpdate() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	result, err := selfupdate.Update(ctx, version)
	if err != nil {
		return err
	}
	if result.Deferred {
		fmt.Println("Downloaded ccauth " + result.Version + ". Windows will finish the update after this process exits.")
		return nil
	}
	if result.Version == version {
		fmt.Printf("Already up to date: ccauth %s\n", result.Version)
		return nil
	}
	fmt.Printf("Updated to ccauth %s\n", result.Version)
	return nil
}

func last(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

func joinScopes(sc []string) string {
	out := ""
	for i, s := range sc {
		if i > 0 {
			out += ", "
		}
		out += s
	}
	return out
}

func settingsStatus(p store.Profile) string {
	if !p.Settings.Included {
		return "not saved"
	}
	return fmt.Sprintf("saved (%d bytes)", len(p.Settings.Data))
}

func offerUpdate(ctx context.Context, args []string) bool {
	if os.Getenv("CCAUTH_NO_UPDATE_CHECK") != "" || !selfupdate.IsReleaseVersion(version) || skipUpdateCheck(args) {
		return false
	}
	checkCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	info, err := selfupdate.Check(checkCtx, version, false)
	cancel()
	if err != nil || !info.Available {
		return false
	}
	fmt.Fprintf(os.Stderr, "A new ccauth version is available: %s -> %s\n", version, info.LatestVersion)
	if info.URL != "" {
		fmt.Fprintln(os.Stderr, info.URL)
	}
	if !isTerminal(os.Stdin) || !isTerminal(os.Stdout) || !isTerminal(os.Stderr) {
		fmt.Fprintln(os.Stderr, "Run 'ccauth update' to install it.")
		return false
	}
	fmt.Fprint(os.Stderr, "Update now before running the command? [Y/n] ")
	answer, readErr := bufio.NewReader(os.Stdin).ReadString('\n')
	if readErr != nil && len(answer) == 0 {
		fmt.Fprintln(os.Stderr)
		return false
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	if answer != "" && answer != "y" && answer != "yes" && answer != "д" && answer != "да" {
		return false
	}
	updateCtx, updateCancel := context.WithTimeout(ctx, 5*time.Minute)
	result, err := selfupdate.Update(updateCtx, version)
	updateCancel()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ccauth: automatic update failed: %v\nContinuing with the current version.\n", err)
		return false
	}
	if result.Deferred {
		fmt.Fprintf(os.Stderr, "Downloaded ccauth %s. Windows will finish the update after this process exits.\nRun the command again.\n", result.Version)
	} else if result.Version == version {
		fmt.Fprintf(os.Stderr, "Already up to date: ccauth %s\n", result.Version)
	} else {
		fmt.Fprintf(os.Stderr, "Updated to ccauth %s. Run the command again.\n", result.Version)
	}
	return true
}

func skipUpdateCheck(args []string) bool {
	for _, arg := range args {
		switch arg {
		case "help", "-h", "--help", "--version", "-v", "version":
			return true
		}
	}
	return false
}

func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
