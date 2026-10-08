package terminal

import (
	"io"
	"strings"
)

// Writer sends sequences to the terminal, wrapping them for tmux passthrough
// (requires `set -g allow-passthrough on`) when needed.
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
		if w.Tmux {
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

func tmuxWrap(s []byte) []byte {
	return []byte(esc + "Ptmux;" + strings.ReplaceAll(string(s), esc, esc+esc) + st)
}
