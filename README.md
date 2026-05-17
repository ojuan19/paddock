# paddock

> Multi-Claude control plane. Auto-switch Claude Code accounts per directory.
> Color-coded statusline so you never wonder which Claude is talking to you.

## Quickstart

```bash
npm install -g paddockcli                       # any platform with Node
# or:  brew install ojuan19/tap/paddock         # macOS / Linux Homebrew
paddock init                                    # adopt your existing ~/.claude/ as 'personal'
paddock add work --color amber                  # blank profile; you'll log in on first use
cd ~/work && paddock link work                  # bind this directory to the work profile
eval "$(paddock shell-init zsh)" >> ~/.zshrc    # one-time: enable auto-switch on cd
```

Now `cd ~/work && claude` uses your work account; `cd` anywhere else falls back to personal. The statusline at the bottom of Claude tells you which is active.

## Status

Building toward **v0.1.0** — install via build from source today; Homebrew tap + npm wrapper land with v0.1.0 this week.

**Works today (10 commands)**:
- `paddock init` — first-run setup; imports `~/.claude/` as a profile
- `paddock add <name> [--color X] [--default] [--from <dir>]` — create profile (optionally import existing CLAUDE_CONFIG_DIR)
- `paddock list` (`ls`, `--links`) — list profiles
- `paddock link [<name>]` / `paddock unlink` — bind/unbind current dir
- `paddock use <name> [args...]` — launch claude with a specific profile
- `paddock run [args...]` — launch claude with the profile resolved from cwd
- `paddock which [--quiet]` — show which profile applies and why
- `paddock shell-init [bash|zsh|fish|powershell]` — print shell hook script
- `paddock doctor [--fix]` — diagnose installation; auto-repair safe issues

**Other capabilities**:
- Auto-switch on `cd` via shell hook (bash, zsh, fish; powershell experimental)
- Color-coded statusline inside Claude Code per profile
- Allowlist-based import (no caches, history, or session state carried over)
- Fuzzy "did you mean" suggestions on profile typos
- `NO_COLOR` / `FORCE_COLOR` env vars respected

## How it works

Claude Code respects the `CLAUDE_CONFIG_DIR` environment variable. Pointing it at a different directory gives complete isolation — credentials, settings, history, plugins, MCPs all switch. On macOS, Keychain entries are scoped by the config dir's path, so credentials stay separate per profile.

Paddock orchestrates this:
1. You create profiles (`paddock add <name>` or `paddock init`)
2. You link directories to profiles (`paddock link <name>`)
3. A shell hook (`paddock shell-init`) sets `CLAUDE_CONFIG_DIR` whenever you `cd`
4. A statusline command (`paddock statusline`) shows the active profile *inside* Claude Code

Paddock does **not** modify Claude Code, does **not** touch `~/.claude/` (except read-only on `init`), and does **not** handle API keys (subscriptions only). Uninstall paddock and Claude Code works exactly as before.

## Install

```bash
# npm (recommended for JS/TS devs)
npm install -g paddockcli

# Homebrew
brew install ojuan19/tap/paddock

# curl | bash (Linux/macOS)
curl -fsSL https://raw.githubusercontent.com/ojuan19/paddock/main/install.sh | bash
```

### Build from source (contributors)

```bash
git clone https://github.com/ojuan19/paddock.git
cd paddock
go build -o paddock ./cmd/paddock
sudo cp paddock /usr/local/bin/
paddock --version
```

Requires Go 1.21+.

### Shell setup

After install, enable auto-switching:

```bash
# zsh
eval "$(paddock shell-init zsh)" >> ~/.zshrc

# bash
eval "$(paddock shell-init bash)" >> ~/.bashrc

# fish
paddock shell-init fish | source
```

## Migration from direnv or manual `CLAUDE_CONFIG_DIR` setups

Already DIY'd a second Claude account using direnv + `.envrc`? One command:

```bash
paddock init                                                  # imports ~/.claude/ → 'personal'
paddock add work --color amber --from ~/.claude-work          # imports your second setup → 'work'
cd ~/work-dir && paddock link work
rm ~/work-dir/.envrc                                          # only if direnv was managing the switch
```

`--from` runs the same allowlist-based copy as `init` (skips caches, history, session state).

> Note: blank profiles created with `paddock add` (no `--from`) require `/login` on first use — Claude Code's auth lives in macOS Keychain keyed by config-dir path, so each new profile path starts unauthenticated.

## Roadmap

- **Rounds 0–7.5** (done) — All 10 commands, auto-switch, statusline, doctor, fuzzy match, `--from` import
- **Round 8** ✅ — tests (resolver + config; shell tests already exist)
- **Round 9** ✅ — release infra (GoReleaser, Homebrew tap, npm wrapper, `curl | bash`)
- **v0.2** — `paddock costs` (per-profile spend), `paddock sync-mcp` (copy MCPs between profiles), `paddock snapshot create/restore`
- **v0.3+** — TUI, team profiles (shared config via git)

## Prior art

- [direnv](https://direnv.net/) — the conceptual ancestor. Paddock is direnv specialized for Claude Code with a visible statusline and one-command UX. If you're already happy with direnv + `.envrc`, paddock's `v0.1` doesn't unlock much — but `v0.2` will, with cost tracking and MCP syncing direnv literally cannot do.
- [`claude-profile`](https://github.com/yu-iskw/claude-profile) — similar idea, no auto-switch, no statusline.

## Contributing

Project is alpha; opening it up for issues + PRs after v0.1 ships. Star the repo to follow along.

## License

MIT — see [LICENSE](LICENSE).
