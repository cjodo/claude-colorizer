# Regenerates docs/demo.gif. Run from the repo root after `make build`:
#   vhs docs/demo/demo.tape
#
# docs/demo/session.sh plays a mock Claude Code session through every state
# (idle → working → attention → done → error), firing the real hooks and
# statusline along the way.

Output docs/demo.gif

Set Shell "bash"
Set FontSize 18
Set Width 1100
Set Height 480
Set Padding 24
Set Theme "Catppuccin Mocha"
Set Framerate 30

# Isolate from the host: default config and a throwaway state dir.
Hide
Type `export PATH="$PWD/bin:$PATH" XDG_CONFIG_HOME="$(mktemp -d)" CLAUDE_COLORIZER_STATE_DIR="$(mktemp -d)" CLAUDE_COLORIZER_TERMINAL=generic PS1='' && clear`
Enter
Type "docs/demo/session.sh"
Enter
Show

Sleep 21s
