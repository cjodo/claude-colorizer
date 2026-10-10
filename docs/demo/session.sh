#!/usr/bin/env bash
# Plays a scripted Claude Code session for docs/demo.gif. Each step fires the
# real hook (`claude-colorizer hook`), which tints the background and records
# the state, then redraws a mock Claude Code screen whose last line is the
# real statusline output. Run by docs/demo/demo.tape; needs claude-colorizer
# on $PATH and CLAUDE_COLORIZER_STATE_DIR set.
set -euo pipefail

SESSION=demo
TRANSCRIPT=$(mktemp)

ESC=$'\e'
R="$ESC[0m" DIM="$ESC[2m" BOLD="$ESC[1m"
ORANGE="$ESC[38;2;215;119;87m" GREY="$ESC[38;2;136;136;136m"
RED="$ESC[38;2;239;68;68m" GREEN="$ESC[38;2;34;197;94m"
COLS=$(tput cols)
STATE=""

# Each state's tab color, from the effective config, for the mock tab strip.
declare -A TAB
while read -r state hex; do TAB[$state]=$hex; done < <(
	claude-colorizer config | awk '/^    "[a-z]+": [{]/ { gsub(/[":{ ]/, ""); s = $0 }
		s && /"tab"/ { gsub(/[",]/, ""); print s, $2; s = "" }')

rgb() { printf '%d;%d;%d' "0x${1:1:2}" "0x${1:3:2}" "0x${1:5:2}"; }

hook() { # EVENT [NOTIFICATION_TYPE MESSAGE]
	case $1 in
	SessionStart) STATE=idle ;; UserPromptSubmit | PostToolUse) STATE=working ;;
	Notification) STATE=attention ;; Stop) STATE=done ;; *Failure) STATE=error ;;
	esac
	printf '{"hook_event_name":"%s","session_id":"%s","cwd":"%s","notification_type":"%s","message":"%s"}' \
		"$1" "$SESSION" "$PWD" "${2:-}" "${3:-}" | claude-colorizer hook
}

say() { # TEXT: append an assistant message, so its colors reach the swatches
	printf '{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"%s"}]}}\n' "$1" >>"$TRANSCRIPT"
}

hr() { local s; printf -v s '%*s' "$((COLS - 2))" ''; printf '%s' "${s// /─}"; }

convo="" input="" spinner=""

# draw repaints the whole screen in one write, so no frame is half drawn.
draw() {
	local box status tabs
	tabs="$ESC[48;2;$(rgb "${TAB[$STATE]}")m$ESC[38;2;17;17;27m$BOLD  Claude Code · my-app  $R$DIM   zsh   $R"
	box="$GREY╭$(hr)╮$R"$'\n'
	box+="$GREY│$R > $input$(printf '%*s' "$((COLS - 5 - ${#input}))" '')$GREY│$R"$'\n'
	box+="$GREY╰$(hr)╯$R"
	status=$(printf '{"session_id":"%s","transcript_path":"%s"}' "$SESSION" "$TRANSCRIPT" | claude-colorizer statusline)
	printf '%s' "$ESC[H$ESC[2J$tabs"$'\n\n'"$ORANGE✻$R ${BOLD}Claude Code$R $DIM· ~/repos/my-app$R"$'\n\n'"$convo$spinner"$'\n'"$box"$'\n'" $status"
}

add() { convo+="$1"$'\n'; }

type_prompt() { # TEXT
	local i
	for ((i = 1; i <= ${#1}; i++)); do
		input=${1:0:i}
		draw
		sleep 0.05
	done
	sleep 0.4
	input=""
	add "$GREY>$R $1"$'\n'
	printf '{"type":"user","message":{"role":"user","content":"%s"}}\n' "$1" >>"$TRANSCRIPT"
}

think() { # SECONDS LABEL
	local frames=(· ✢ ✳ ✶ ✻ ✽) i
	for ((i = 0; i < $1 * 8; i++)); do
		spinner="$ORANGE${frames[i % 6]} $2…$R"$'\n'
		draw
		sleep 0.125
	done
	spinner=""
}

tput civis
trap 'tput cnorm; rm -f "$TRANSCRIPT"' EXIT

# idle: the session just started.
hook SessionStart
draw
sleep 1.6

# working: a prompt was sent, Claude reads a file.
type_prompt "make the header warmer"
hook UserPromptSubmit
think 1 Thinking
add "$ORANGE⏺$R ${BOLD}Read$R(src/header.css)"
add "  $GREY⎿  Read 42 lines$R"$'\n'
hook PostToolUse
think 1 Thinking

# attention: a permission prompt waits on the user.
spinner="$ORANGE⏺$R ${BOLD}Update$R(src/header.css)"$'\n'"  ${BOLD}Do you want to make this edit to header.css?$R"$'\n'"  $ORANGE❯ 1. Yes$R"$'\n'"    2. No"$'\n'
hook Notification permission_prompt "Claude needs your permission to use Edit"
draw
sleep 2.2
spinner=""

# working again once approved, then done.
add "$ORANGE⏺$R ${BOLD}Update$R(src/header.css)"
add "  $GREY⎿  Updated with 3 changes$R"$'\n'
hook PostToolUse
think 1 Writing
say "Warmed the header: #f38ba8 title, #fab387 accents, #f9e2af highlights on #1e1e2e."
add "$ORANGE⏺$R Warmed the header: ${BOLD}#f38ba8$R title, ${BOLD}#fab387$R accents and"
add "  ${BOLD}#f9e2af$R highlights on ${BOLD}#1e1e2e$R."$'\n'
hook Stop
draw
sleep 2.4

# error: the next prompt runs a failing tool.
convo=""
type_prompt "run the tests"
hook UserPromptSubmit
think 1 Running
add "$ORANGE⏺$R ${BOLD}Bash$R(npm test)"
add "  $GREY⎿  $RED✗ header.test.ts: expected contrast ≥ 4.5, got 3.9$R"$'\n'
hook PostToolUseFailure
draw
sleep 2.4
