//go:build windows

package terminal

import "os"

// OpenTTY opens the attached console's output buffer. Windows Terminal
// interprets VT sequences written there.
func OpenTTY() (*os.File, error) {
	return os.OpenFile("CONOUT$", os.O_WRONLY, 0)
}
