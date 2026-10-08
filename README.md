# claude-colorizer

Color visualization for Claude Code, in the spirit of
[nvim-colorizer](https://github.com/NvChad/nvim-colorizer.lua). Written in Go
with no dependencies.

- **Statusline swatches.** Colors mentioned in the conversation (Claude's
  replies, code it writes, your prompts) appear as live swatches under the
  prompt. It reads `#rgb`, `#rrggbb(aa)`, `rgb()/rgba()`, `hsl()/hsla()` and
  `oklch()`. Turn them off with `"statusline": {"swatches": false}`.
- **State colors.** The terminal tab and background tint change as the
  session moves through *idle → working → needs you → done / error*.
- **State indicator.** The current state also appears as a colored dot at
  the start of the statusline (`● working`), on the line Claude Code already
  reserves under the prompt. It works in any terminal and isn't overwritten
  the way the title glyph can be.
- **`show` / `try`.** Highlight color literals in any file
  (`claude-colorizer show styles.css`), or preview a color as your terminal
  background (`claude-colorizer try '#1e1e2e'`).

## Terminal support

Ghostty is the most tested terminal: it's the one claude-colorizer is
developed in. The other drivers follow each terminal's documented escape
sequences and have unit tests, but see less real use. If you use one of
them, see [CONTRIBUTING.md](CONTRIBUTING.md) for a short test checklist and
how to report results or add a terminal.

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

**Program status (OSC 7501).** Every state change is also reported with the
[Program Status Protocol](https://www.superlogical.com/rex/docs/build/program-status),
which is not tied to any terminal. Terminals that implement it (Ghostty,
through libghostty) can show the session state natively, with no glyph or
config. The others ignore it. The report carries `app=claude-code`, the title
`Claude Code · <dir>`, and for permission prompts `state=blocked:kind=permission`
plus Claude's notification message. Inside tmux it needs passthrough enabled
(see [tmux](#tmux)). To turn it off, set `"status": false`.

Each terminal is a driver implementing `terminal.Terminal`
(`internal/terminal/drivers.go`). Detection reads environment variables
(`internal/terminal/detect.go`). Inside tmux it asks tmux which terminal
each client tty attached to the session is running instead, because the
environment describes the terminal that started the tmux server. If
different terminals are attached, tab colors are sent to all of them. To override detection, set
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

**Title fallback**: Claude Code sets the terminal title itself and may
overwrite the glyph. If your version supports
`CLAUDE_CODE_DISABLE_TERMINAL_TITLE=1`, set it. If you don't want the glyph,
set `"title": false`.

### tmux

Background tint and title work without any setup, because tmux understands
those sequences and applies them to the pane. Everything else has to reach
the outer terminal through tmux passthrough:

- tab colors (Kitty, iTerm2, WezTerm)
- program status reports (OSC 7501)

claude-colorizer wraps these for passthrough automatically when `$TMUX` is
set, but tmux drops them unless passthrough is on. Add this to `~/.tmux.conf`
(or `~/.config/tmux/tmux.conf`):

```tmux
set -g allow-passthrough all
```

Reload it in a running tmux with `tmux source-file ~/.tmux.conf`.

| Value | Effect                                                              |
|-------|---------------------------------------------------------------------|
| `off` | Default. Tab colors and status reports are dropped.                 |
| `on`  | Passed through only from panes currently visible.                   |
| `all` | Passed through from every pane, including windows you aren't viewing. |

Use `all` so a Claude session in a background window can still mark its tab
as blocked or done while you are elsewhere. `on` drops those updates until you
switch back to the window. The tradeoff is that any program in any pane can
then send escape sequences straight to the outer terminal.

To try it in the current pane only, without editing your config:

```sh
tmux set -p allow-passthrough all
```

Check the global value with `tmux show -gv allow-passthrough`, and the
detected terminal with `claude-colorizer detect` (it prints `tmux: yes`).

**Background opacity.** tmux paints a tinted pane with an explicit color in
every cell, and terminals normally draw explicit cell colors fully opaque, so
a translucent terminal turns solid while the tint is on. There are two fixes:

- **Keep the tint on Claude's pane** and have the terminal apply its opacity
  to explicit cell colors too. In Ghostty 1.2+, add this to its config:

  ```
  background-opacity-cells = true
  ```

  Other explicitly colored cells (the tmux status bar, editor themes) then
  turn translucent as well.

- **Tint the whole terminal instead**, for terminals without such an option:

  ```json
  { "tmuxBackground": "terminal" }
  ```

  This needs passthrough enabled, and the tint covers every pane in the
  window rather than just Claude's. The default is `"pane"`.

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
- **Refresh:** the statusline gets `"refreshInterval": 2`. Permission prompts
  and idle notifications don't make Claude Code re-run the statusline, so
  without it the state indicator would lag until the next message.
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

The repo is also a plugin marketplace. The plugin provides the hooks and
inline highlighting (below), but not the statusline, because plugins can't
set one:

```
/plugin marketplace add /path/to/claude-colorizer
/plugin install claude-colorizer@claude-colorizer
```

Then run `claude-colorizer install --no-hooks` to add the statusline without
registering the hooks twice. To try it for a single session without installing,
run `claude --plugin-dir /path/to/claude-colorizer`.

### Inline highlighting

Inline highlighting draws each color literal in Claude's replies on its own
color, like nvim-colorizer does in a buffer. Replies without colors are drawn
as usual. In a reply with colors, lines holding a color are drawn as plain
text, so their bold and inline code markers are dropped.

**It only works when the repo is loaded as a plugin.** The highlighter is a
function-hooks module (`hooks/colorize.tsx`, with `hooks/colors.ts` porting
`internal/colors`), and Claude Code loads it only from the plugin's
`hooks/hooks.json`. `make install` writes plain command hooks to
`settings.json`, which can't load modules, so with `make install` alone
you get state colors and swatches but no inline highlighting.

To turn it on, load the plugin in either of these ways:

- For one session: `claude --plugin-dir /path/to/claude-colorizer`.
- Permanently: install it as shown in
  [Alternative: as a Claude Code plugin](#alternative-as-a-claude-code-plugin).
  Installed plugins run from a cached copy, so reinstall after changing
  `hooks/`.

If you already ran `make install`, switch its hooks over to the plugin so
each event doesn't fire twice:

```sh
claude-colorizer uninstall && claude-colorizer install --no-hooks
```

Highlighting starts in the next session. To check that it loaded, ask Claude
to print a hex color such as `#ff5733`; it should appear on an orange
background. Run its tests with `claude plugin test .`.

## What the colors mean

The tab, background tint and title glyph show what the session is doing, so
you can tell from another tab or window whether Claude needs you.

| Color         | Glyph | State     | Meaning                                                    |
|---------------|-------|-----------|------------------------------------------------------------|
| Slate         | ⚪    | idle      | The session just started (or was cleared) and is waiting for your first prompt. |
| Blue          | 🔵    | working   | Claude is working: thinking, writing, running tools. Nothing for you to do yet. |
| Amber         | 🟡    | attention | Claude is waiting on you: a permission prompt, or it has sat idle waiting for input. |
| Green         | 🟢    | done      | Claude finished its turn. Read the reply and send the next prompt. |
| Red           | 🔴    | error     | Something failed: the API request errored, or a tool call failed. |
| Your defaults | none  | reset     | The session ended.                                         |

Red after a failed tool call doesn't always mean the turn is over. Claude
often recovers, and the color goes back to blue on its next successful tool
call. Red after an API error stays until you send another prompt.

The glyph is a fallback for terminals that can't color their tabs. It appears
at the start of the window title. See the Title fallback column in
[Terminal support](#terminal-support).

### Hook events behind each state

| Hook event                          | State     | Default tab / tint      | OSC 7501 `state` |
|-------------------------------------|-----------|-------------------------|------------------|
| `UserPromptSubmit`, `PostToolUse`   | working   | `#3b82f6` / `#151b2b`   | `working`        |
| `Notification` (permission, idle)   | attention | `#f59e0b` / `#2a2112`   | `blocked`        |
| `Stop`                              | done      | `#22c55e` / `#13231a`   | `done`           |
| `StopFailure`, `PostToolUseFailure` | error     | `#ef4444` / `#2b1515`   | `error`          |
| `SessionStart`                      | idle      | `#94a3b8` / `#181b21`   | `idle`           |
| `SessionEnd`                        | reset     | terminal defaults       | `clear`          |

To change any color or glyph, see [Configuration](#configuration).

## Configuration

`~/.config/claude-colorizer/config.json` (or `$CLAUDE_COLORIZER_CONFIG`). Every
key is optional and is layered over the defaults. Run `claude-colorizer config`
to print the effective config.

```json
{
  "tab": true,
  "background": true,
  "title": true,
  "status": true,
  "tmuxBackground": "pane",
  "states": {
    "working": { "tab": "#7c3aed", "background": "#1a1426" },
    "done":    { "background": "#eef9f0" }
  },
  "statusline": {
    "swatches": true,
    "max": 12,
    "sources": ["assistant", "tools", "user"],
    "label": "hex",
    "indicator": "label",
    "prefix": "🎨 ",
    "empty": ""
  }
}
```

`tmuxBackground` only matters inside tmux: `"pane"` (default) tints just
Claude's pane, `"terminal"` tints the outer terminal and keeps its background
opacity (see [tmux](#tmux)).

`statusline.swatches` turns the color swatches on (default) or off. With
`false`, the transcript isn't read and the statusline shows only the state
indicator. The other swatch keys (`max`, `sources`, `label`, `prefix`,
`empty`) then have no effect.

`statusline.indicator` controls the state indicator: `"label"` (default,
`● working`), `"dot"` (just `●`), or `"none"`. The dot uses the state's `tab`
color. Hooks record each session's state in a small file under your cache
directory (`~/.cache/claude-colorizer/sessions` on Linux, or
`$CLAUDE_COLORIZER_STATE_DIR`), which is deleted when the session ends.

The default background tints assume a dark theme. On a light theme, set
light `background` values, or set `"background": false`.

## Development

```sh
make test     # go vet + unit tests (parser, drivers, detection, transcript)
make cross    # linux/darwin/windows binaries in dist/
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for testing on a terminal and adding
a new one.
