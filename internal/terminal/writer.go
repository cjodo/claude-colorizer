package terminal

import (
	"io"
	"strings"
)

// Writer sends sequences to the terminal. Inside tmux, sequences tmux
// understands itself (background, text color, title) are sent as-is so tmux applies them
// to the pane; anything else is wrapped for passthrough to the outer terminal
// (requires `set -g allow-passthrough on`).
type Writer struct {
	W    io.Writer
	Tmux bool
	// PassBackground passes background and text color sequences through to the outer
	// terminal instead of letting tmux apply them to the pane. tmux paints a
	// recolored pane with an explicit cell background, which terminals draw
	// fully opaque; the outer terminal's own default background keeps its
	// background-opacity. The tint then covers the whole window.
	PassBackground bool
}

// Write emits each non-empty sequence. Nil sequences are skipped, so callers
// can pass a driver's result without checking capabilities first.
func (w Writer) Write(seqs ...[]byte) error {
	var b []byte
	for _, s := range seqs {
		if len(s) == 0 {
			continue
		}
		switch {
		case w.Tmux && w.PassBackground && isPaneColor(s):
			// A reset also clears any pane color tmux holds from before,
			// so the pane goes back to drawing the default.
			if strings.HasPrefix(string(s), esc+"]111") || strings.HasPrefix(string(s), esc+"]110") {
				b = append(b, s...)
			}
			s = tmuxWrap(s)
		case w.Tmux && !tmuxNative(s):
			s = tmuxWrap(s)
		}
		b = append(b, s...)
	}
	if len(b) == 0 {
		return nil
	}
	_, err := w.W.Write(b)
	return err
}

// tmuxNative reports whether tmux handles s itself: OSC 10/110 and 11/111
// set and reset the pane text color and background, OSC 0/2 set the pane
// title. Passing these through
// would bypass tmux and recolor the whole outer window instead of the pane.
func tmuxNative(s []byte) bool {
	if isPaneColor(s) {
		return true
	}
	for _, p := range []string{"0;", "2;"} {
		if strings.HasPrefix(string(s), esc+"]"+p) {
			return true
		}
	}
	return false
}

// isPaneColor reports whether s sets or resets the background (OSC 11/111)
// or text color (OSC 10/110).
func isPaneColor(s []byte) bool {
	for _, p := range []string{"11;", "111" + st, "111" + bel, "10;", "110" + st, "110" + bel} {
		if strings.HasPrefix(string(s), esc+"]"+p) {
			return true
		}
	}
	return false
}

func tmuxWrap(s []byte) []byte {
	return []byte(esc + "Ptmux;" + strings.ReplaceAll(string(s), esc, esc+esc) + st)
}
