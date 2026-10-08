# Contributing

Thanks for helping out. The most useful contributions right now are
**reports and fixes for terminals other than Ghostty**.

## Where things stand

claude-colorizer is developed and tested day to day in **Ghostty** on Linux,
both bare and inside tmux. That's the setup to trust.

The other drivers (kitty, WezTerm, iTerm2, Warp, Alacritty, Windows Terminal)
are written from each terminal's documented escape sequences and covered by
unit tests, but they get much less real-world use. If you run one of them,
a quick pass through [Testing on your terminal](#testing-on-your-terminal) and
a report of what worked is a real contribution, even if everything works.

## Setup

You need Go 1.22+ and Claude Code.

```sh
git clone <this repo> && cd claude-colorizer
make test      # go vet + unit tests
make build     # binary in bin/
make install   # go install + register statusline and hooks
```

## Testing on your terminal

Test in two stages. Stage 1 checks the driver by hand. Stage 2 checks the
hooks inside a real Claude Code session. If stage 1 fails, the driver or
detection is wrong. If only stage 2 fails, the problem is in how the hooks
reach the terminal.

### 1. The driver, outside Claude Code

```sh
claude-colorizer detect            # right terminal, expected capabilities?
claude-colorizer set idle          # slate tab / tint, ⚪ title glyph
claude-colorizer set working       # blue, 🔵
claude-colorizer set attention     # amber
claude-colorizer set done          # green
claude-colorizer set error         # red
claude-colorizer reset             # back to your own theme
claude-colorizer try '#2a2112'     # background preview
echo 'a #5fb38a b hsl(160,12%,58%)' | claude-colorizer show
```

Check that:

- `detect` names your terminal. If it says `generic`, see
  [Adding or fixing detection](#adding-or-fixing-detection).
- Each `set` changes what the [support table](README.md#terminal-support)
  says it should: tab color, background tint, or the title glyph.
- `reset` returns to **your theme's** colors, not to plain black or white.
- `show` highlights both colors in truecolor.

To tell a detection bug from a driver bug, force a driver:

```sh
CLAUDE_COLORIZER_TERMINAL=kitty claude-colorizer set working
```

### 2. The same, inside tmux

Repeat stage 1 in a tmux pane. `detect` should still name the outer
terminal. The background should change for that pane only. Tab colors for
kitty, iTerm2 and WezTerm, and OSC 7501 status reports, also need
`tmux set -g allow-passthrough all` (see [tmux](README.md#tmux)).

### 3. A real Claude Code session

```sh
CLAUDE_COLORIZER_DEBUG=1 claude
```

| Do this                                    | Expect                          |
|--------------------------------------------|---------------------------------|
| Start `claude` (or run `/clear`)           | slate (idle)                    |
| Send any prompt                            | blue (working)                  |
| Ask for a command that needs permission    | amber (attention)               |
| Let the reply finish                       | green (done)                    |
| Ask it to run `false`                      | red, then blue on the next tool |
| Ask for a color palette                    | swatches under the prompt       |
| `/exit`                                    | your terminal's own colors      |

`CLAUDE_COLORIZER_DEBUG=1` prints hook errors to stderr.

### Common failures

| Symptom                                        | Likely cause                                                       |
|------------------------------------------------|--------------------------------------------------------------------|
| `detect` says `generic`                        | The terminal's env var didn't make it through (SSH, `sudo`, tmux)  |
| Works by hand, not inside Claude Code          | The hook can't reach the TTY; check the debug output               |
| Colors stay after quitting                     | `SessionEnd` didn't run (the session was killed or crashed)        |
| Swatches look washed out or wrong              | No truecolor; `echo $COLORTERM` should print `truecolor`           |

## Reporting a terminal

Open an issue with:

- Terminal name, version and OS, and whether tmux was involved (with its
  version and `allow-passthrough` setting).
- The output of `claude-colorizer detect`.
- The output of `env | grep -iE 'term|kitty|ghostty|wezterm|iterm|warp|alacritty|wt_session|tmux'`.
  Check it for anything private before you paste it.
- Which rows of the stage 1 and stage 3 checklists worked and which didn't.

A report that says "everything works in X" is welcome too: it lets us mark
that terminal as tested.

## Adding a new terminal

All terminal-specific code lives in `internal/terminal/`. A driver only
**builds byte sequences**; `Writer` delivers them to the TTY and wraps them
for tmux. That keeps drivers pure and easy to test.

### 1. Write the driver

Add a type to `internal/terminal/drivers.go` that implements
`terminal.Terminal`. Embed the shared helpers rather than rewriting them:

| Helper  | Use it when the terminal...                        |
|---------|----------------------------------------------------|
| `osc11` | sets and resets the background with OSC 11 / 111   |
| `titled`| sets the title with OSC 2                          |
| `noTab` | has no escape sequence for tab color               |

```go
// Foot: OSC 11/111 and titles; no tabs.

type Foot struct {
	osc11
	noTab
	titled
}

func (Foot) Name() string { return "foot" }
func (Foot) Capabilities() Capabilities {
	return Capabilities{Background: true, Title: true, TrueColor: true}
}
```

Rules:

- A method returns `nil` when the terminal can't do that action.
  `Capabilities()` must agree: `TestCapabilitiesMatchSequences` fails if a
  capability is true but its method returns `nil`, or the other way round.
- Use ST (`ESC \`) as the string terminator, via `osc()`. Use BEL only where
  the terminal is known to require it, and say so in a comment.
- Pass any user-controlled text through `sanitize` so it can't end the
  sequence early.
- Start the block with a comment naming the sequences used and any
  configuration the user needs (like kitty's `allow_remote_control`).

### 2. Register it

- Add it to `All()` in `internal/terminal/terminal.go`. The order there is
  the order shown by `detect`.
- Add its name to the `CLAUDE_COLORIZER_TERMINAL` list in the usage text in
  `cmd/claude-colorizer/main.go`.

### Adding or fixing detection

`Detect` in `internal/terminal/detect.go` reads environment variables:

- `TERM_PROGRAM` is checked first, but tmux overwrites it with `tmux`.
- The terminal's **own** variables (`KITTY_WINDOW_ID`, `WEZTERM_PANE`, ...)
  are checked next, because they survive into tmux. Prefer one of those
  for a new terminal.
- Anything unrecognized falls through to `Generic`.

### 3. Test it

In `internal/terminal/terminal_test.go`:

- Add `TestDetect` cases for each variable you detect on, including one
  with `TERM_PROGRAM=tmux` if the terminal is usable inside tmux.
- Add a `TestSequences` case with the exact bytes for any sequence that
  isn't shared with another driver.

Then run `make test` and the manual checklist above in the real terminal.

### 4. Document it

- Add a row to the [support table](README.md#terminal-support) in the README,
  with a footnote for any setup the user needs.
- Mention in the PR which checklist rows you verified by hand.

## Pull requests

- Keep each PR to one change: one driver, one fix, or one doc update.
- `make test` must pass.
- For a terminal change, say which terminal, version and OS you tested on,
  and whether you tested inside tmux.
