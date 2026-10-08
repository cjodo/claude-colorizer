# Regenerates docs/demo.gif. Run from the repo root after `make build`:
#   vhs docs/demo/demo.tape

Output docs/demo.gif

Set Shell "bash"
Set FontSize 18
Set Width 1100
Set Height 640
Set Padding 24
Set Theme "Catppuccin Mocha"
Set TypingSpeed 40ms

# Isolate from the host: default config, and a "done" session for the indicator.
Hide
Type `export PATH="$PWD/bin:$PATH" XDG_CONFIG_HOME="$(mktemp -d)" CLAUDE_COLORIZER_STATE_DIR="$(mktemp -d)" PS1='$ '`
Enter
Type `printf done > "$CLAUDE_COLORIZER_STATE_DIR/demo" && clear`
Enter
Show

Type "claude-colorizer show docs/demo/sample.css"
Sleep 300ms
Enter
Sleep 2s

Type "claude-colorizer statusline < docs/demo/status.json"
Sleep 300ms
Enter
Sleep 3s
