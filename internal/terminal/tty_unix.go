//go:build !windows

package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// OpenTTY opens the controlling terminal for writing. Hook and statusline
// processes have their stdout captured by Claude Code, so escape sequences
// must bypass it. If this process has no controlling terminal (e.g. it was
// started in a new session), the terminal of the nearest ancestor that has
// one — the Claude Code process itself — is used instead.
func OpenTTY() (*os.File, error) {
	if f, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0); err == nil {
		return f, nil
	}
	pid := os.Getppid()
	for range 8 {
		if pid <= 1 {
			break
		}
		if dev := ttyOf(pid); dev != "" {
			return os.OpenFile(dev, os.O_WRONLY, 0)
		}
		pid = parentOf(pid)
	}
	return nil, fmt.Errorf("no terminal found for this process or its ancestors")
}

func ttyOf(pid int) string {
	if _, err := os.Stat("/proc/self"); err == nil {
		for _, fd := range []string{"0", "1", "2"} {
			dst, err := os.Readlink(filepath.Join("/proc", strconv.Itoa(pid), "fd", fd))
			if err == nil && (strings.HasPrefix(dst, "/dev/pts/") || strings.HasPrefix(dst, "/dev/tty")) {
				return dst
			}
		}
		return ""
	}
	out, err := exec.Command("ps", "-o", "tty=", "-p", strconv.Itoa(pid)).Output()
	tty := strings.TrimSpace(string(out))
	if err != nil || tty == "" || tty == "?" || tty == "??" {
		return ""
	}
	if !strings.HasPrefix(tty, "/dev/") {
		tty = "/dev/" + tty
	}
	return tty
}

func parentOf(pid int) int {
	if b, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat")); err == nil {
		// Fields after the parenthesized command name: state, ppid, ...
		if i := strings.LastIndexByte(string(b), ')'); i >= 0 {
			if f := strings.Fields(string(b)[i+1:]); len(f) > 1 {
				p, _ := strconv.Atoi(f[1])
				return p
			}
		}
		return 0
	}
	out, err := exec.Command("ps", "-o", "ppid=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return 0
	}
	p, _ := strconv.Atoi(strings.TrimSpace(string(out)))
	return p
}
