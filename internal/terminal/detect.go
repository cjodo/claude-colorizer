package terminal

import "strings"

// EnvOverride forces a driver by name, bypassing detection.
const EnvOverride = "CLAUDE_COLORIZER_TERMINAL"

// Detect picks a driver from the environment. getenv is os.Getenv in
// production. Terminal-specific variables are checked before TERM_PROGRAM
// because they survive into tmux, where TERM_PROGRAM becomes "tmux".
func Detect(getenv func(string) string) Terminal {
	if name := getenv(EnvOverride); name != "" {
		if t, ok := ByName(name); ok {
			return t
		}
	}

	term := getenv("TERM")
	switch strings.ToLower(getenv("TERM_PROGRAM")) {
	case "ghostty":
		return Ghostty{}
	case "wezterm":
		return WezTerm{}
	case "iterm.app":
		return ITerm2{}
	case "warpterminal":
		return Warp{}
	}

	switch {
	case getenv("KITTY_WINDOW_ID") != "" || term == "xterm-kitty":
		return Kitty{}
	case getenv("GHOSTTY_RESOURCES_DIR") != "" || term == "xterm-ghostty":
		return Ghostty{}
	case getenv("WEZTERM_PANE") != "" || getenv("WEZTERM_EXECUTABLE") != "":
		return WezTerm{}
	case getenv("ITERM_SESSION_ID") != "" || getenv("LC_TERMINAL") == "iTerm2":
		return ITerm2{}
	case getenv("WARP_IS_LOCAL_SHELL_SESSION") != "":
		return Warp{}
	case getenv("ALACRITTY_WINDOW_ID") != "" || getenv("ALACRITTY_SOCKET") != "" || term == "alacritty":
		return Alacritty{}
	case getenv("WT_SESSION") != "":
		return WindowsTerminal{}
	}
	return Generic{}
}

// InTmux reports whether sequences must be wrapped for tmux passthrough.
func InTmux(getenv func(string) string) bool { return getenv("TMUX") != "" }
