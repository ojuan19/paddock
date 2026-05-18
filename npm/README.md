# paddockcli

Multi-Claude Code account manager. Auto-switch per directory.

This is the npm wrapper. Full docs, examples, and source live at:
**https://github.com/ojuan19/paddock**

## Install

```bash
npm install -g paddockcli
```

This downloads the matching paddock binary from GitHub Releases on install.

## Example

```console
$ paddock init
Imported ~/.claude/ as profile 'personal' (blue, default)

$ paddock add work --from ~/.claude-work --color amber
Created profile 'work' (amber)

$ cd ~/work && paddock link work
Linked work
Shell auto-switch not installed. Add to ~/.zshrc? [Y/n] y
Added paddock hook to ~/.zshrc.

$ claude       # uses the work account automatically; statusline shows [work]
```

After this, `cd`'ing into a linked directory auto-switches the Claude account.

## License

MIT
