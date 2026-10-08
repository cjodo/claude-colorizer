package terminal

import (
	"os/exec"
	"strings"
)

// EnvOverride forces a driver by name, bypassing detection.
const EnvOverride = "CLAUDE_COLORIZER_TERMINAL"

// Detect picks a driver from the environment. getenv is os.Getenv in
// production.
//
// Inside tmux, the environment describes whichever terminal started the tmux
// server, which may not be the one attached now. So the terminal of every
// client tty attached to this pane's session is asked for instead, and the
// environment is only a fallback.
func Detect(getenv func(string) string) Terminal {
	if name := getenv(EnvOverride); name != "" {
		if t, ok := ByName(name); ok {
			return t
		}
	}
	if InTmux(getenv) {
		if t := fromClients(tmuxClients(getenv("TMUX_PANE"))); t != nil {
			return t
		}
	}
	return fromEnv(getenv)
}

// tmuxClients returns "termtype|termname" for each client tty attached to
// the session containing pane. A variable so tests can stub it.
var tmuxClients = func(pane string) []string {
	args := []string{"list-clients", "-F", "#{client_termtype}|#{client_termname}"}
	if pane != "" {
		args = append(args, "-t", pane)
	}
	out, err := exec.Command("tmux", args...).Output()
	if err != nil {
		return nil
	}
	return strings.Fields(strings.ReplaceAll(strings.TrimSpace(string(out)), " ", "_"))
}

// fromClients builds a driver covering every attached client, or nil if
// there are none.
func fromClients(clients []string) Terminal {
	var ts []Terminal
	seen := map[string]bool{}
	for _, c := range clients {
		t := fromTermType(c)
		if !seen[t.Name()] {
			seen[t.Name()] = true
			ts = append(ts, t)
		}
	}
	switch len(ts) {
	case 0:
		return nil
	case 1:
		return ts[0]
	}
	return Multi(ts)
}

// fromTermType maps tmux's client_termtype/client_termname (e.g.
// "kitty(0.49.2)|xterm-kitty", "ghostty_1.3.1|xterm-ghostty") to a driver.
func fromTermType(s string) Terminal {
	s = strings.ToLower(s)
	switch {
	case strings.Contains(s, "kitty"):
		return Kitty{}
	case strings.Contains(s, "ghostty"):
		return Ghostty{}
	case strings.Contains(s, "wezterm"):
		return WezTerm{}
	case strings.Contains(s, "iterm2"):
		return ITerm2{}
	case strings.Contains(s, "alacritty"):
		return Alacritty{}
	}
	return Generic{}
}

// fromEnv detects the terminal from its environment variables.
// Terminal-specific variables are checked after TERM_PROGRAM because they
// survive into tmux, where TERM_PROGRAM becomes "tmux".
func fromEnv(getenv func(string) string) Terminal {
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

// InTmux reports whether output goes through tmux (see Writer).
func InTmux(getenv func(string) string) bool { return getenv("TMUX") != "" }
