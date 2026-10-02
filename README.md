# ccauth

Save/restore Claude Code authorization profiles: named profiles with OAuth tokens,
account metadata and (optionally) `settings.json`. Shows token expiry.

## What it stores

- OAuth credentials (`claudeAiOauth`: access + refresh tokens, expiry, scopes)
  — read from macOS Keychain (`Claude Code-credentials`) or
  `~/.claude/.credentials.json` on Linux/WSL; honors `CLAUDE_CONFIG_DIR`
- Account metadata from `~/.claude.json` (`oauthAccount`: email, org, uuid, tier)
- `~/.claude/settings.json` — only with `--settings`

Profiles live in `~/.ccauth/profiles/<name>.json` with `0600` permissions
(override dir with `CCAUTH_DIR`).

## Commands

```
ccauth save <name> [--settings]    # snapshot current auth
ccauth restore <name> [--settings] # switch to profile (alias: use)
ccauth list                        # profiles + expiry, * = active
ccauth show <name>                 # details
ccauth current                     # live credentials info
ccauth remove <name>               # delete profile
ccauth update                      # update to the latest release
```

Example:

```
$ ccauth save work --settings
$ ccauth save personal --settings
$ ccauth list
   NAME       EMAIL/ORG                    SUBSCRIPTION  ACCESS EXPIRES                REFRESH EXPIRES                SETTINGS
*  work       me@corp.com / Corp           pro           2026-10-02 18:10 (in 6h 17m)  2026-10-29 02:24 (in 26d 14h)  yes
   personal   me@gmail.com / -             pro           ...                           ...                            yes
$ ccauth restore personal
$ ccauth restore work --settings   # also swaps settings.json (backup: settings.json.ccauth-bak)
```

## Install

macOS / Linux:

```bash
curl -fsSL https://raw.githubusercontent.com/Shooa/ccauth/main/install.sh | sh
```

Windows (PowerShell):

```powershell
irm https://raw.githubusercontent.com/Shooa/ccauth/main/install.ps1 | iex
```

Or manually: grab `ccauth_<ver>_<os>_<arch>.tar.gz` from
[GitHub Releases](https://github.com/Shooa/ccauth/releases) and put the binary on `PATH`.

## Auto-update

On every command `ccauth` checks GitHub Releases (at most once per 24h, 3s timeout,
result cached). If a newer version exists it asks `Update now? [Y/n]` and replaces
the binary in place (download verified against `SHA256SUMS`). Disable with
`CCAUTH_NO_UPDATE_CHECK=1`. Explicit: `ccauth update`.

## Security notes

- Refresh tokens are long-lived — profile files are effectively full account access; keep `~/.ccauth` private (permissions enforced at `0600`/`0700`).
- Restoring on macOS writes the Keychain (via `security add-generic-password -U`); the on-disk `.credentials.json` is refreshed only if it already exists, to avoid stale-file shadowing.
- Restart running Claude Code sessions after `restore`.
