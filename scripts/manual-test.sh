#!/bin/sh
# Manual checks for the parts unit tests can't see: what actually reaches the
# terminal. Builds the binary from the working tree first, so it tests your
# changes rather than whatever is installed. See CONTRIBUTING.md.
#
#   scripts/manual-test.sh hooks    hook events -> session state -> statusline (automatic, silent)
#   scripts/manual-test.sh states   cycle every state on this terminal (watch it)
#   scripts/manual-test.sh tmux     passthrough setup, raw OSC 7501, background window,
#                                   tmuxBackground=terminal
#   scripts/manual-test.sh herdr    detection, pane title read back from herdr, pane tint
#   scripts/manual-test.sh all      all of the above; tmux and herdr each run only
#                                   when this shell is inside them
#
# DELAY sets the seconds each state is shown (default 2). CLAUDE_COLORIZER_CONFIG
# is honored by `states`, `tmux` and `herdr`, so you can preview your own palette.
set -eu

root="$(cd "$(dirname "$0")/.." && pwd)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
bin="$work/claude-colorizer"
delay="${DELAY:-2}"
esc="$(printf '\033')"
fails=0

(cd "$root" && go build -o "$bin" ./cmd/claude-colorizer)

say()  { printf '\n\033[1m== %s\033[0m\n' "$*"; }
ok()   { printf '  \033[32mok\033[0m    %s\n' "$*"; }
bad()  { printf '  \033[31mFAIL\033[0m  %s\n' "$*"; fails=$((fails + 1)); }
note() { printf '        %s\n' "$*"; }

# hooks: drives each hook event with a throwaway session id and state dir, with
# every terminal output disabled, and checks the saved state and the statusline.
check_hooks() {
  say "hooks -> session state -> statusline"
  printf '{"tab":false,"background":false,"title":false,"status":false}' >"$work/quiet.json"
  dir="$work/sessions"
  sid="manual-test"

  run_hook() { # event [extra json fields] [session id]
    printf '{"hook_event_name":"%s","session_id":"%s","cwd":"%s"%s}' "$1" "${3:-$sid}" "$root" "${2:-}" |
      CLAUDE_COLORIZER_CONFIG="$work/quiet.json" CLAUDE_COLORIZER_STATE_DIR="$dir" CLAUDE_COLORIZER_DEBUG=1 "$bin" hook
  }
  statusline() { # [config]
    printf '{"session_id":"%s"}' "$sid" |
      CLAUDE_COLORIZER_CONFIG="${1:-$work/quiet.json}" CLAUDE_COLORIZER_STATE_DIR="$dir" "$bin" statusline |
      sed "s/${esc}\[[0-9;]*m//g"
  }
  expect() { # event want-state want-line [extra]
    run_hook "$1" "${4:-}"
    got_state="$(cat "$dir/$sid" 2>/dev/null || echo '<none>')"
    got_line="$(statusline)"
    if [ "$got_state" = "$2" ] && [ "$got_line" = "$3" ]; then
      ok "$(printf '%-17s %-10s %s' "$1" "$got_state" "$got_line")"
    else
      bad "$1: state=$got_state line='$got_line', want state=$2 line='$3'"
    fi
  }

  expect SessionStart     idle      "● idle"
  expect UserPromptSubmit working   "● working"
  expect Notification     attention "● attention" ',"notification_type":"permission_prompt","message":"needs permission"'
  expect Stop             done      "● done"
  expect StopFailure      error     "● error"
  expect PostToolUse      working   "● working"
  expect SessionEnd       '<none>'  ""

  echo working >"$dir/$sid"
  for mode in label:"● working" dot:"●" none:""; do
    printf '{"tab":false,"background":false,"title":false,"status":false,"statusline":{"indicator":"%s"}}' "${mode%%:*}" >"$work/mode.json"
    got="$(statusline "$work/mode.json")"
    [ "$got" = "${mode#*:}" ] && ok "indicator ${mode%%:*}: '$got'" || bad "indicator ${mode%%:*}: '$got', want '${mode#*:}'"
  done

  printf '%s\n' '{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"try #3b82f6"}]}}' >"$work/t.jsonl"
  for mode in true:"● working     #3b82f6" false:"● working"; do
    printf '{"tab":false,"background":false,"title":false,"status":false,"statusline":{"swatches":%s}}' "${mode%%:*}" >"$work/mode.json"
    got="$(printf '{"session_id":"%s","transcript_path":"%s"}' "$sid" "$work/t.jsonl" |
      CLAUDE_COLORIZER_CONFIG="$work/mode.json" CLAUDE_COLORIZER_STATE_DIR="$dir" "$bin" statusline |
      sed "s/${esc}\[[0-9;]*m//g")"
    [ "$got" = "${mode#*:}" ] && ok "swatches ${mode%%:*}: '$got'" || bad "swatches ${mode%%:*}: '$got', want '${mode#*:}'"
  done

  echo bogus >"$dir/$sid"
  [ -z "$(statusline)" ] && ok "unknown state shows nothing" || bad "unknown state rendered '$(statusline)'"

  run_hook UserPromptSubmit "" "../escape"
  [ ! -e "$work/escape" ] && ok "session id '../escape' rejected" || bad "session id '../escape' wrote outside the state dir"
}

# states: the stage 1 checklist from CONTRIBUTING.md, on this terminal.
check_states() {
  say "states on this terminal (watch the tab, background and title)"
  "$bin" detect | sed 's/^/        /'
  in_tmux && note "inside tmux: see the tmux stage for what needs passthrough"
  in_herdr && note "inside herdr: expect the pane tint and title, never a tab color"
  for s in idle working attention done error; do
    note "$s"
    "$bin" set "$s"
    sleep "$delay"
  done
  "$bin" reset
  note "reset: your own theme should be back"
}

in_tmux()  { [ -n "${TMUX:-}" ]; }
in_herdr() { [ "${HERDR_ENV:-}" = 1 ] || [ "${TERM_PROGRAM:-}" = herdr ]; }

# tmux: passthrough setup, a raw OSC 7501 report, and one from a hidden window.
check_tmux() {
  say "tmux passthrough"
  if ! in_tmux; then
    note "not inside tmux; skipped"
    return
  fi
  live="$(tmux show -gv allow-passthrough 2>/dev/null || echo '?')"
  case "$live" in
    all) ok "allow-passthrough is 'all'" ;;
    on)  ok "allow-passthrough is 'on'"; note "reports from windows you aren't viewing are dropped; use 'all'" ;;
    *)   bad "allow-passthrough is '$live' in the running server"
         for f in "$HOME/.tmux.conf" "${XDG_CONFIG_HOME:-$HOME/.config}/tmux/tmux.conf"; do
           grep -qs 'allow-passthrough' "$f" && note "$f sets it; reload with: tmux source-file $f"
         done
         return ;;
  esac

  note "raw OSC 7501 'working' for ${delay}s, bypassing claude-colorizer"
  printf '\033Ptmux;\033\033]7501;state=working:app=manual-test\033\033\\\033\\'
  sleep "$delay"
  printf '\033Ptmux;\033\033]7501;state=clear\033\033\\\033\\'
  note "did the terminal show it? if not, but passthrough is on, the terminal ignores OSC 7501"

  note "'attention' from a hidden window for $((delay * 2))s"
  tmux new-window -d "'$bin' set attention; sleep $((delay * 2)); '$bin' reset"
  sleep $((delay * 2 + 1))
  note "it should have reached the terminal although that window was never visible"

  printf '{"tab":false,"title":false,"status":false,"tmuxBackground":"terminal"}' >"$work/opacity.json"
  note "'working' tint with tmuxBackground=terminal for ${delay}s"
  CLAUDE_COLORIZER_CONFIG="$work/opacity.json" "$bin" set working
  sleep "$delay"
  CLAUDE_COLORIZER_CONFIG="$work/opacity.json" "$bin" reset
  note "the whole window should have been tinted, keeping any background opacity"
}

# herdr: herdr emulates its panes and has no passthrough, so only the pane
# background and pane title can work. The title is read back from herdr.
check_herdr() {
  say "herdr"
  if ! in_herdr; then
    note "not inside herdr; skipped"
    return
  fi
  got="$("$bin" detect | sed -n 's/^terminal: *//p')"
  [ "$got" = herdr ] && ok "detected as herdr" || bad "detected as '$got', want 'herdr'"

  pane_title() {
    herdr pane get "${HERDR_PANE_ID:-}" 2>/dev/null | sed -n 's/.*"terminal_title":"\([^"]*\)".*/\1/p'
  }
  if in_tmux; then
    note "tmux runs inside this herdr pane: the title goes to tmux, and tab colors and"
    note "status reports stop at herdr; the tmux stage covers the rest"
  elif [ -n "${CLAUDECODE:-}" ]; then
    note "running under Claude Code, which keeps rewriting the pane title; run from a plain shell to check it"
  elif [ -z "${HERDR_PANE_ID:-}" ] || ! command -v herdr >/dev/null 2>&1; then
    note "HERDR_PANE_ID or the herdr CLI is missing; can't read the pane title back"
  else
    want="Claude Code · $(basename "$PWD")"
    printf '{"tab":false,"background":false,"status":false}' >"$work/title.json"
    CLAUDE_COLORIZER_CONFIG="$work/title.json" "$bin" set working
    sleep 1
    got="$(pane_title)"
    case "$got" in
      ?*" $want") ok "pane title after 'working': '$got'" ;;
      *)          bad "pane title after 'working' is '$got', want '<glyph> $want'" ;;
    esac
    CLAUDE_COLORIZER_CONFIG="$work/title.json" "$bin" reset
    sleep 1
    got="$(pane_title)"
    # reset has no session cwd, so the title loses the directory too.
    [ "$got" = "Claude Code" ] && ok "pane title after reset: '$got'" || bad "pane title after reset is '$got', want 'Claude Code'"
  fi

  note "'working' tint on this pane for ${delay}s"
  "$bin" set working
  sleep "$delay"
  "$bin" reset
  note "only this pane should have been tinted, not the whole window or other panes"
}

case "${1:-all}" in
  hooks)  check_hooks ;;
  states) check_states ;;
  tmux)   check_tmux ;;
  herdr)  check_herdr ;;
  all)    check_hooks; check_states; check_tmux; check_herdr ;;
  *)      sed -n '6,12p' "$0" | sed 's/^# //' >&2; exit 2 ;;
esac

if [ "$fails" -gt 0 ]; then
  say "$fails check(s) failed"
  exit 1
fi
