package terminal

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cjodo/claude-colorizer/internal/colors"
)

// osc11 is embedded by drivers that use the standard OSC 11 / OSC 111 pair
// for the background.
type osc11 struct{}

func (osc11) SetBackground(c colors.Color) []byte { return oscSetBackground(c) }
func (osc11) ResetBackground() []byte             { return oscResetBackground() }

// noTab is embedded by drivers with no escape sequence for tab color.
type noTab struct{}

func (noTab) SetTabColor(colors.Color) []byte { return nil }
func (noTab) ResetTabColor() []byte           { return nil }

// titled is embedded by drivers that support OSC 2.
type titled struct{}

func (titled) SetTitle(t string) []byte { return oscTitle(t) }

// ---------------------------------------------------------------------------
// Kitty: tab color through the remote-control protocol sent in-band over the
// TTY. Requires `allow_remote_control yes` in kitty.conf; otherwise kitty
// ignores the request and only background/title apply.

type Kitty struct {
	osc11
	titled
}

func (Kitty) Name() string { return "kitty" }
func (Kitty) Capabilities() Capabilities {
	return Capabilities{TabColor: true, Background: true, Title: true, TrueColor: true}
}

func (Kitty) SetTabColor(c colors.Color) []byte {
	v := int(c.R)<<16 | int(c.G)<<8 | int(c.B)
	return kittyCmd("set-tab-color", map[string]any{
		"self":   true,
		"colors": map[string]any{"active_bg": v, "inactive_bg": v, "active_fg": fgInt(c), "inactive_fg": fgInt(c)},
	})
}

func (Kitty) ResetTabColor() []byte {
	return kittyCmd("set-tab-color", map[string]any{
		"self":   true,
		"colors": map[string]any{"active_bg": nil, "inactive_bg": nil, "active_fg": nil, "inactive_fg": nil},
	})
}

func fgInt(c colors.Color) int {
	f := c.Contrast()
	return int(f.R)<<16 | int(f.G)<<8 | int(f.B)
}

func kittyCmd(cmd string, payload map[string]any) []byte {
	body, _ := json.Marshal(map[string]any{
		"cmd":         cmd,
		"version":     []int{0, 35, 0},
		"no_response": true,
		"payload":     payload,
	})
	return []byte(esc + "P@kitty-cmd" + string(body) + st)
}

// ---------------------------------------------------------------------------
// Ghostty: OSC 11/111 and titles. No tab-color sequence exists.

type Ghostty struct {
	osc11
	noTab
	titled
}

func (Ghostty) Name() string { return "ghostty" }
func (Ghostty) Capabilities() Capabilities {
	return Capabilities{Background: true, Title: true, TrueColor: true}
}

// ---------------------------------------------------------------------------
// WezTerm: no direct tab-color sequence, so the color is published as the
// user variable `claude_colorizer_tab` (OSC 1337 SetUserVar) and a
// format-tab-title handler in wezterm.lua paints the tab. See README.

type WezTerm struct {
	osc11
	titled
}

const WezTermUserVar = "claude_colorizer_tab"

func (WezTerm) Name() string { return "wezterm" }
func (WezTerm) Capabilities() Capabilities {
	return Capabilities{TabColor: true, Background: true, Title: true, TrueColor: true}
}
func (WezTerm) SetTabColor(c colors.Color) []byte { return weztermUserVar(WezTermUserVar, c.Hex()) }
func (WezTerm) ResetTabColor() []byte             { return weztermUserVar(WezTermUserVar, "") }

func weztermUserVar(name, value string) []byte {
	return []byte(esc + "]1337;SetUserVar=" + name + "=" + base64.StdEncoding.EncodeToString([]byte(value)) + bel)
}

// ---------------------------------------------------------------------------
// iTerm2: proprietary OSC 6 tab color; OSC 1337 SetColors for background.

type ITerm2 struct{ titled }

func (ITerm2) Name() string { return "iterm2" }
func (ITerm2) Capabilities() Capabilities {
	return Capabilities{TabColor: true, Background: true, Title: true, TrueColor: true}
}

func (ITerm2) SetTabColor(c colors.Color) []byte {
	return []byte(fmt.Sprintf(
		esc+"]6;1;bg;red;brightness;%d"+bel+esc+"]6;1;bg;green;brightness;%d"+bel+esc+"]6;1;bg;blue;brightness;%d"+bel,
		c.R, c.G, c.B))
}
func (ITerm2) ResetTabColor() []byte { return []byte(esc + "]6;1;bg;*;default" + bel) }

func (ITerm2) SetBackground(c colors.Color) []byte {
	return []byte(esc + "]1337;SetColors=bg=" + c.Hex()[1:] + bel)
}
func (ITerm2) ResetBackground() []byte { return oscResetBackground() }

// ---------------------------------------------------------------------------
// Warp: renders truecolor and honors titles, but background escapes are not
// reliably supported and tab colors are UI-only.

type Warp struct {
	noTab
	titled
}

func (Warp) Name() string                      { return "warp" }
func (Warp) Capabilities() Capabilities        { return Capabilities{Title: true, TrueColor: true} }
func (Warp) SetBackground(colors.Color) []byte { return nil }
func (Warp) ResetBackground() []byte           { return nil }

// ---------------------------------------------------------------------------
// Alacritty: OSC 11/111 and titles; it has no tabs.

type Alacritty struct {
	osc11
	noTab
	titled
}

func (Alacritty) Name() string { return "alacritty" }
func (Alacritty) Capabilities() Capabilities {
	return Capabilities{Background: true, Title: true, TrueColor: true}
}

// ---------------------------------------------------------------------------
// Windows Terminal: OSC 11/111 and titles. Tab color is settings-only.

type WindowsTerminal struct {
	osc11
	noTab
	titled
}

func (WindowsTerminal) Name() string { return "windows-terminal" }
func (WindowsTerminal) Capabilities() Capabilities {
	return Capabilities{Background: true, Title: true, TrueColor: true}
}

// ---------------------------------------------------------------------------
// Generic: unknown xterm-compatible terminal. Uses only the most widely
// implemented sequences.

type Generic struct {
	osc11
	noTab
	titled
}

func (Generic) Name() string { return "generic" }
func (Generic) Capabilities() Capabilities {
	return Capabilities{Background: true, Title: true, TrueColor: true}
}

// ---------------------------------------------------------------------------
// Multi: several different terminals attached to one tmux session. Tab
// colors are sent in every driver's dialect (each terminal ignores the
// others'); background and title come from the first driver that has them,
// since tmux applies those to the pane once for all clients.

type Multi []Terminal

func (m Multi) Name() string {
	names := make([]string, len(m))
	for i, t := range m {
		names[i] = t.Name()
	}
	return strings.Join(names, "+")
}

func (m Multi) Capabilities() Capabilities {
	var c Capabilities
	for _, t := range m {
		tc := t.Capabilities()
		c.TabColor = c.TabColor || tc.TabColor
		c.Background = c.Background || tc.Background
		c.Title = c.Title || tc.Title
		c.TrueColor = c.TrueColor || tc.TrueColor
	}
	return c
}

func (m Multi) SetTabColor(c colors.Color) []byte {
	return m.all(func(t Terminal) []byte { return t.SetTabColor(c) })
}
func (m Multi) ResetTabColor() []byte { return m.all(Terminal.ResetTabColor) }

func (m Multi) SetBackground(c colors.Color) []byte {
	return m.first(func(t Terminal) []byte { return t.SetBackground(c) })
}
func (m Multi) ResetBackground() []byte { return m.first(Terminal.ResetBackground) }
func (m Multi) SetTitle(s string) []byte {
	return m.first(func(t Terminal) []byte { return t.SetTitle(s) })
}

func (m Multi) all(f func(Terminal) []byte) []byte {
	var b []byte
	for _, t := range m {
		b = append(b, f(t)...)
	}
	return b
}

func (m Multi) first(f func(Terminal) []byte) []byte {
	for _, t := range m {
		if s := f(t); s != nil {
			return s
		}
	}
	return nil
}
