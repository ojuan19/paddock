# paddock

> Multi-Claude control plane. Auto-switch Claude Code accounts per directory.
> Color-coded statusline so you never wonder which Claude is talking to you.

```
$ paddock ls
● personal     12 projects
● work         3 projects   ← in clientX subtree
● acme         1 project

$ cd ~/work/clientX
[work] Claude switched.
```

---

## 🚧 Status: alpha — building in public

Active development. **Not ready to install yet** — release infra ships at the end of v0.1.

If you use Claude Code with more than one account today, the canonical workaround is `direnv` + `.envrc` files pointing `CLAUDE_CONFIG_DIR` per directory. Paddock is that workflow specialized: one CLI, auto-discovery, a statusline you can see inside Claude, and (coming soon) cost tracking per profile.

### What works today

```bash
paddock add work --color amber
paddock add personal --color blue --default
paddock list
```

Config stored atomically in `~/.paddock/config.json`. That's it for now — no switching yet. The interesting part lands in the next few commits.

### Roadmap

- ✅ **v0.0 (now)** — Scaffolding: `add`, `list`, config storage
- 🚧 **v0.1 (this weekend)** — Auto-switch + release infra
  - `paddock run` / `paddock which` / `paddock link` / `paddock unlink`
  - `paddock shell-init` (bash/zsh/fish auto-switch on `cd`)
  - Color-coded statusline inside Claude Code
  - `paddock doctor` (diagnostics for issue reports)
  - Homebrew tap + npm wrapper + `curl | bash` installer
- 🔮 **v0.2 (week after)** — The killer features
  - `paddock costs` — per-profile cost breakdown (token data is already in `stats-cache.json`)
  - `paddock sync-mcp` — copy MCP servers between profiles
  - `paddock snapshot create/restore` — backup config before risky changes
- 🔮 **v0.3+** — TUI, team profiles (shared config via git)

---

## How it works

Claude Code respects the `CLAUDE_CONFIG_DIR` environment variable. Pointing it at a different directory gives complete isolation — credentials, settings, history, plugins, MCPs all switch. On macOS, Keychain entries are scoped by the config dir's path hash, so credentials stay separate per profile.

Paddock orchestrates this:
1. You create profiles (`paddock add <name>`)
2. You link directories to profiles (`paddock link <name>` — coming v0.1)
3. A shell hook (`paddock shell-init`) sets `CLAUDE_CONFIG_DIR` whenever you `cd`
4. A statusline command (`paddock statusline`) shows the active profile *inside* Claude Code

Paddock does **not** modify Claude Code, does **not** touch `~/.claude/`, and does **not** handle API keys (subscriptions only). Uninstall paddock and Claude Code works exactly as before.

## Prior art

- [direnv](https://direnv.net/) — the conceptual ancestor. Paddock is direnv specialized for Claude Code with a visible statusline and one-command UX. If you're already happy with direnv + `.envrc`, you don't need paddock yet — but you might want it when `costs` and `sync-mcp` ship in v0.2.
- [`claude-profile`](https://github.com/yu-iskw/claude-profile) — similar idea, no auto-switch, no statusline.

## Install

Not yet. When v0.1 ships:

```bash
# npm (any platform with Node)
npm install -g paddockcli

# Homebrew (macOS / Linux)
brew install ojuan19/tap/paddock

# Or just download a binary
curl -fsSL https://raw.githubusercontent.com/ojuan19/paddock/main/install.sh | bash
```

## Building from source

```bash
git clone https://github.com/ojuan19/paddock.git
cd paddock
go build -o paddock ./cmd/paddock
./paddock --version
```

Requires Go 1.21+.

## Contributing

Project is alpha and I'm working through a tight build plan in rounds (a few commits per round, then break). Issues and PRs welcome after v0.1 ships. Until then, star ⭐ the repo to follow along.

## License

MIT — see [LICENSE](LICENSE).
