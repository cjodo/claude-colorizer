package terminal

import (
	"io"
	"strings"
)

// Writer sends sequences to the terminal. Inside tmux, sequences tmux
// understands itself (background, title) are sent as-is so tmux applies them
// to the pane; anything else is wrapped for passthrough to the outer terminal
// (requires `set -g allow-passthrough on`).
type Writer struct {
	W    io.Writer
	Tmux bool
}

// Write emits each non-empty sequence. Nil sequences are skipped, so callers
// can pass a driver's result without checking capabilities first.
func (w Writer) Write(seqs ...[]byte) error {
	var b []byte
	for _, s := range seqs {
		if len(s) == 0 {
			continue
		}
		if w.Tmux && !tmuxNative(s) {
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

// tmuxNative reports whether tmux handles s itself: OSC 11/111 set and reset
// the pane background, OSC 0/2 set the pane title. Passing these through
// would bypass tmux and recolor the whole outer window instead of the pane.
func tmuxNative(s []byte) bool {
	for _, p := range []string{"11;", "111" + st, "111" + bel, "0;", "2;"} {
		if strings.HasPrefix(string(s), esc+"]"+p) {
			return true
		}
	}
	return false
}

func tmuxWrap(s []byte) []byte {
	return []byte(esc + "Ptmux;" + strings.ReplaceAll(string(s), esc, esc+esc) + st)
}
