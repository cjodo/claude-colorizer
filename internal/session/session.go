// Package session remembers each Claude Code session's current state, so the
// statusline (a separate process) can show what the hooks last applied.
package session

import (
	"os"
	"path/filepath"
	"strings"
)

// EnvDir overrides the directory state files are kept in.
const EnvDir = "CLAUDE_COLORIZER_STATE_DIR"

// Dir returns where per-session state files live.
func Dir() string {
	if d := os.Getenv(EnvDir); d != "" {
		return d
	}
	if d, err := os.UserCacheDir(); err == nil {
		return filepath.Join(d, "claude-colorizer", "sessions")
	}
	return filepath.Join(os.TempDir(), "claude-colorizer-sessions")
}

// Save records state for session; an empty state forgets the session.
func Save(session, state string) error {
	p, ok := path(session)
	if !ok {
		return nil
	}
	if state == "" {
		err := os.Remove(p)
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	// Write then rename, so a concurrent statusline never reads half a file.
	f, err := os.CreateTemp(filepath.Dir(p), ".tmp-*")
	if err != nil {
		return err
	}
	_, err = f.WriteString(state)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(f.Name(), p)
	}
	if err != nil {
		os.Remove(f.Name())
	}
	return err
}

// Load returns the last state saved for session, or "" if none.
func Load(session string) string {
	p, ok := path(session)
	if !ok {
		return ""
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// path maps a session id to its file, rejecting ids that could escape Dir.
func path(session string) (string, bool) {
	if session == "" || len(session) > 128 {
		return "", false
	}
	for _, c := range session {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return "", false
		}
	}
	return filepath.Join(Dir(), session), true
}
