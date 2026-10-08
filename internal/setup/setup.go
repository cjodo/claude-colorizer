// Package setup wires claude-colorizer into Claude Code's settings.json
// (statusLine + hooks) and removes it again.
package setup

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Events the hook subcommand reacts to; tool events need a matcher.
var hookEvents = []struct {
	Name    string
	Matcher bool
}{
	{"SessionStart", false},
	{"UserPromptSubmit", false},
	{"PostToolUse", true},
	{"PostToolUseFailure", true},
	{"Notification", false},
	{"Stop", false},
	{"StopFailure", false},
	{"SessionEnd", false},
}

// marker identifies entries we own, regardless of the binary's path.
const marker = "claude-colorizer"

type Options struct {
	Settings   string // path to settings.json
	Binary     string // absolute path to the claude-colorizer binary
	StateDir   string // where the previous statusLine is saved for uninstall
	Hooks      bool
	Statusline bool
	DryRun     bool
}

// DefaultSettingsPath is the user-level Claude Code settings file.
func DefaultSettingsPath() string {
	dir := os.Getenv("CLAUDE_CONFIG_DIR")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".claude")
	}
	return filepath.Join(dir, "settings.json")
}

// Install adds the statusLine and hooks. It is idempotent and returns a
// human-readable list of changes.
func Install(o Options) ([]string, error) {
	s, err := load(o.Settings)
	if err != nil {
		return nil, err
	}
	var changes []string
	bin := shellQuote(o.Binary)

	if o.Statusline {
		cmd := bin + " statusline"
		prev, _ := s["statusLine"].(map[string]any)
		prevCmd, _ := prev["command"].(string)
		switch {
		case strings.Contains(prevCmd, marker):
			changes = append(changes, "statusLine: already installed")
		default:
			if prevCmd != "" {
				// Keep the user's statusline: it renders above the swatches.
				cmd += " -- sh -c " + shellQuote(prevCmd)
				if !o.DryRun {
					if err := savePrevious(o.StateDir, prev); err != nil {
						return nil, err
					}
				}
				changes = append(changes, "statusLine: chained existing command "+prevCmd)
			} else {
				changes = append(changes, "statusLine: added")
			}
			s["statusLine"] = map[string]any{"type": "command", "command": cmd, "padding": 0}
		}
	}

	if o.Hooks {
		hooks, _ := s["hooks"].(map[string]any)
		if hooks == nil {
			hooks = map[string]any{}
		}
		added := 0
		for _, ev := range hookEvents {
			groups, _ := hooks[ev.Name].([]any)
			if containsMarker(groups) {
				continue
			}
			g := map[string]any{"hooks": []any{map[string]any{
				"type": "command", "command": bin + " hook", "timeout": 5,
			}}}
			if ev.Matcher {
				g["matcher"] = "*"
			}
			hooks[ev.Name] = append(groups, g)
			added++
		}
		s["hooks"] = hooks
		if added > 0 {
			changes = append(changes, fmt.Sprintf("hooks: added %d events", added))
		} else {
			changes = append(changes, "hooks: already installed")
		}
	}

	if o.DryRun {
		return changes, nil
	}
	return changes, save(o.Settings, s)
}

// Uninstall removes everything Install added and restores a chained
// statusLine.
func Uninstall(o Options) ([]string, error) {
	s, err := load(o.Settings)
	if err != nil {
		return nil, err
	}
	var changes []string

	if sl, _ := s["statusLine"].(map[string]any); sl != nil {
		if cmd, _ := sl["command"].(string); strings.Contains(cmd, marker) {
			if prev, ok := loadPrevious(o.StateDir); ok {
				s["statusLine"] = prev
				changes = append(changes, "statusLine: restored previous command")
			} else {
				delete(s, "statusLine")
				changes = append(changes, "statusLine: removed")
			}
		}
	}

	if hooks, _ := s["hooks"].(map[string]any); hooks != nil {
		removed := 0
		for name, v := range hooks {
			groups, _ := v.([]any)
			var keep []any
			for _, g := range groups {
				if containsMarker([]any{g}) {
					removed++
					continue
				}
				keep = append(keep, g)
			}
			if len(keep) == 0 {
				delete(hooks, name)
			} else {
				hooks[name] = keep
			}
		}
		if len(hooks) == 0 {
			delete(s, "hooks")
		}
		if removed > 0 {
			changes = append(changes, fmt.Sprintf("hooks: removed %d entries", removed))
		}
	}

	if len(changes) == 0 {
		return []string{"nothing to remove"}, nil
	}
	if o.DryRun {
		return changes, nil
	}
	if err := save(o.Settings, s); err != nil {
		return nil, err
	}
	os.Remove(filepath.Join(o.StateDir, "previous-statusline.json"))
	return changes, nil
}

func containsMarker(groups []any) bool {
	for _, g := range groups {
		gm, _ := g.(map[string]any)
		hs, _ := gm["hooks"].([]any)
		for _, h := range hs {
			hm, _ := h.(map[string]any)
			if cmd, _ := hm["command"].(string); strings.Contains(cmd, marker) {
				return true
			}
		}
	}
	return false
}

func load(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, err
	}
	s := map[string]any{}
	if len(strings.TrimSpace(string(data))) == 0 {
		return s, nil
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("%s is not valid JSON, leaving it untouched: %w", path, err)
	}
	return s, nil
}

// save writes settings atomically, keeping a .bak of the previous file.
func save(path string, s map[string]any) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if old, err := os.ReadFile(path); err == nil {
		if err := os.WriteFile(path+".bak", old, 0o600); err != nil {
			return err
		}
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func savePrevious(dir string, prev map[string]any) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(prev, "", "  ")
	return os.WriteFile(filepath.Join(dir, "previous-statusline.json"), data, 0o600)
}

func loadPrevious(dir string) (map[string]any, bool) {
	data, err := os.ReadFile(filepath.Join(dir, "previous-statusline.json"))
	if err != nil {
		return nil, false
	}
	var m map[string]any
	return m, json.Unmarshal(data, &m) == nil
}

// shellQuote quotes s for POSIX sh, which Claude Code uses to run commands.
func shellQuote(s string) string {
	if s != "" && strings.IndexFunc(s, func(r rune) bool {
		return !(r == '/' || r == '-' || r == '_' || r == '.' || r == ':' ||
			r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z')
	}) < 0 {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
