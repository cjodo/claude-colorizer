# claude-colorizer

Color visualization for Claude Code, in the spirit of
[nvim-colorizer](https://github.com/NvChad/nvim-colorizer.lua). Written in Go
with no dependencies.

- **Statusline swatches.** Colors mentioned in the conversation (Claude's
  replies, code it writes, your prompts) appear as live swatches under the
  prompt. It reads `#rgb`, `#rrggbb(aa)`, `rgb()/rgba()`, `hsl()/hsla()` and
  `oklch()`.
- **State colors.** The terminal tab and background tint change as the
  session moves through *working → needs you → done / error*.
- **`show` / `try`.** Highlight color literals in any file
  (`claude-colorizer show styles.css`), or preview a color as your terminal
  background (`claude-colorizer try '#1e1e2e'`).

## Terminal support

| Terminal         | Tab color                        | Background tint    | Title fallback |
|------------------|----------------------------------|--------------------|----------------|
| Kitty            | ✅ remote control¹               | ✅ OSC 11          | —              |
| Ghostty          | — (no escape exists)             | ✅ OSC 11          | ✅ glyph       |
| WezTerm          | ✅ user var + Lua²               | ✅ OSC 11          | —              |
| iTerm2           | ✅ OSC 6                         | ✅ OSC 1337        | —              |
| Warp             | —                                | —                  | ✅ glyph       |
| Alacritty        | — (no tabs)                      | ✅ OSC 11          | ✅ glyph       |
| Windows Terminal | —                                | ✅ OSC 11          | ✅ glyph       |
| other xterm-like | —                                | ✅ OSC 11          | ✅ glyph       |

Statusline swatches use 24-bit SGR colors, which work in all of them.

Each terminal is a driver implementing `terminal.Terminal`
(`internal/terminal/drivers.go`). Detection reads environment variables
(`internal/terminal/detect.go`). To override detection, set
`CLAUDE_COLORIZER_TERMINAL=kitty|ghostty|wezterm|iterm2|warp|alacritty|windows-terminal|generic`.
Run `claude-colorizer detect` to see what was picked.

¹ **Kitty**: add `allow_remote_control yes` to `kitty.conf` (or `socket-only`
plus `listen_on`). Without it, kitty ignores the tab color and only the
background changes.

² **WezTerm**: the color is published as the user var `claude_colorizer_tab`.
To paint the tab, add this to `wezterm.lua`:

```lua
wezterm.on('format-tab-title', function(tab)
  local c = tab.active_pane.user_vars.claude_colorizer_tab
  if c and c ~= '' then
    return { { Background = { Color = c } }, { Foreground = { Color = '#000000' } },
             { Text = ' ' .. tab.active_pane.title .. ' ' } }
  end
end)
```

**tmux**: background tint and title work without any setup; tmux applies
them to the pane. Tab colors (kitty, iTerm2, WezTerm) have to reach the outer
terminal, so they need `set -g allow-passthrough on`. They are wrapped
automatically when `$TMUX` is set.

**Title fallback**: Claude Code sets the terminal title itself and may
overwrite the glyph. If your version supports
`CLAUDE_CODE_DISABLE_TERMINAL_TITLE=1`, set it. If you don't want the glyph,
set `"title": false`.

## Install

You need Go 1.22+ and Claude Code.

```sh
git clone <this repo> && cd claude-colorizer
make install
```

That's the whole setup. `make install` does two things:

1. Runs `go install`, which builds the binary into `$(go env GOPATH)/bin`.
2. Runs `claude-colorizer install`, which adds the statusline and all hooks to
   `~/.claude/settings.json` using the binary's **absolute path**, so it
   doesn't matter whether that directory is on your `$PATH`.

Restart Claude Code, then try it: ask Claude for a palette and the swatches
appear under the prompt. The tab or background changes while it works.

Tab colors in Kitty and WezTerm need one extra piece of terminal config. See
[Terminal support](#terminal-support).

### What `install` changes

- **Backup:** the previous file is saved as `settings.json.bak`. If
  `settings.json` isn't valid JSON (for example, it has comments), install
  refuses to touch it.
- **Your existing statusline is kept:** it's chained, so its output renders
  on the line above the swatches. `uninstall` puts it back exactly.
- **Hooks:** they are appended next to any hooks you already have, never
  replacing them. Running `install` again doesn't add duplicates.
- **Key order:** keys in `settings.json` are rewritten in alphabetical order.

Preview the changes first with `claude-colorizer install --dry-run`.

| Flag              | Effect                                                   |
|-------------------|----------------------------------------------------------|
| `--dry-run`       | Print the changes without writing                        |
| `--no-hooks`      | Statusline only (or use this if you installed the plugin) |
| `--no-statusline` | State colors only                                        |
| `--settings PATH` | Edit another file, e.g. `.claude/settings.json` in a project |

Pass flags through make with `make install ARGS="--no-statusline"`.

### Uninstall

```sh
make uninstall        # or: claude-colorizer uninstall
```

This removes only the entries it added, restores a chained statusline, and
deletes the binary.

### Alternative: as a Claude Code plugin

The repo is also a plugin marketplace. The plugin only provides the hooks,
because plugins can't set a statusline:

```
/plugin marketplace add /path/to/claude-colorizer
/plugin install claude-colorizer@claude-colorizer
```

Then run `claude-colorizer install --no-hooks` to add the statusline without
registering the hooks twice. To try it for a single session without installing,
run `claude --plugin-dir /path/to/claude-colorizer`.

## State mapping

| Hook event                          | State     | Default tab / tint      |
|-------------------------------------|-----------|-------------------------|
| `UserPromptSubmit`, `PostToolUse`   | working   | `#3b82f6` / `#151b2b`   |
| `Notification` (permission, idle)   | attention | `#f59e0b` / `#2a2112`   |
| `Stop`                              | done      | `#22c55e` / `#13231a`   |
| `StopFailure`, `PostToolUseFailure` | error     | `#ef4444` / `#2b1515`   |
| `SessionStart`, `SessionEnd`        | reset     | terminal defaults       |

## Configuration

`~/.config/claude-colorizer/config.json` (or `$CLAUDE_COLORIZER_CONFIG`). Every
key is optional and is layered over the defaults. Run `claude-colorizer config`
to print the effective config.

```json
{
  "tab": true,
  "background": true,
  "title": true,
  "states": {
    "working": { "tab": "#7c3aed", "background": "#1a1426" },
    "done":    { "background": "#eef9f0" }
  },
  "statusline": {
    "max": 12,
    "sources": ["assistant", "tools", "user"],
    "label": "hex",
    "prefix": "🎨 ",
    "empty": ""
  }
}
```

The default background tints assume a dark theme. On a light theme, set
light `background` values, or set `"background": false`.

## Development

```sh
make test     # go vet + unit tests (parser, drivers, detection, transcript)
make cross    # linux/darwin/windows binaries in dist/
```
