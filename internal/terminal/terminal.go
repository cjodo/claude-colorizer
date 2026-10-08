// Package terminal abstracts the escape sequences each supported terminal
// emulator understands for recoloring tabs, backgrounds and titles.
//
// Implementations only build byte sequences; Writer delivers them to the
// controlling TTY (wrapping for tmux when needed). That keeps every driver
// pure and testable.
package terminal

import (
	"strings"

	"github.com/cjodo/claude-colorizer/internal/colors"
)

// Capabilities describes what a terminal can do natively.
type Capabilities struct {
	// TabColor: the tab (or title bar) can be colored by escape sequence.
	TabColor bool
	// Background: the default background can be set (OSC 11) and reset.
	Background bool
	// Title: the window/tab title can be set (OSC 0/2).
	Title bool
	// TrueColor: 24-bit SGR colors render faithfully.
	TrueColor bool
}

// Terminal is implemented once per supported emulator. A method returns nil
// when the terminal has no way to perform that action.
type Terminal interface {
	Name() string
	Capabilities() Capabilities

	SetTabColor(c colors.Color) []byte
	ResetTabColor() []byte

	SetBackground(c colors.Color) []byte
	ResetBackground() []byte

	SetTitle(title string) []byte
}

// All returns one instance of every driver, in detection-priority order.
func All() []Terminal {
	return []Terminal{
		Kitty{}, Ghostty{}, WezTerm{}, ITerm2{}, Warp{}, Alacritty{}, WindowsTerminal{}, Generic{},
	}
}

// ByName looks up a driver by its Name(), case-insensitively.
func ByName(name string) (Terminal, bool) {
	for _, t := range All() {
		if strings.EqualFold(t.Name(), name) {
			return t, true
		}
	}
	return nil, false
}

// Shared sequence builders. ST (ESC \) is used as the string terminator
// except where a terminal is known to require BEL.

const (
	esc = "\x1b"
	st  = esc + `\`
	bel = "\a"
)

func osc(body string) []byte { return []byte(esc + "]" + body + st) }

func oscSetBackground(c colors.Color) []byte { return osc("11;" + c.Hex()) }

func oscResetBackground() []byte { return osc("111") }

func oscTitle(title string) []byte { return osc("2;" + sanitize(title)) }

// sanitize strips control characters so a title can't terminate the OSC early.
func sanitize(s string) string {
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if c := s[i]; c >= 0x20 && c != 0x7f {
			b = append(b, c)
		}
	}
	return string(b)
}
