// Package config loads ~/.config/claude-colorizer/config.json, layered over
// built-in defaults. Every field is optional.
package config

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// EnvPath overrides the config file location.
const EnvPath = "CLAUDE_COLORIZER_CONFIG"

// State names, driven by Claude Code hook events.
const (
	Working   = "working"
	Attention = "attention"
	Done      = "done"
	Error     = "error"
)

type Style struct {
	Tab        string `json:"tab,omitempty"`        // tab color (hex)
	Background string `json:"background,omitempty"` // background tint (hex)
	Glyph      string `json:"glyph,omitempty"`      // title prefix where tabs can't be colored
}

type Statusline struct {
	Max     int      `json:"max"`     // max swatches shown
	Sources []string `json:"sources"` // any of assistant, tools, user
	Label   string   `json:"label"`   // "hex", "original" or "none"
	Prefix  string   `json:"prefix"`  // printed before the swatches
	Empty   string   `json:"empty"`   // printed when no colors were found
}

type Config struct {
	Tab        *bool            `json:"tab,omitempty"`
	Background *bool            `json:"background,omitempty"`
	Title      *bool            `json:"title,omitempty"`
	States     map[string]Style `json:"states,omitempty"`
	Statusline Statusline       `json:"statusline"`
}

func (c Config) TabEnabled() bool        { return c.Tab == nil || *c.Tab }
func (c Config) BackgroundEnabled() bool { return c.Background == nil || *c.Background }
func (c Config) TitleEnabled() bool      { return c.Title == nil || *c.Title }

// Default tints assume a dark theme; override background colors for light ones.
func Default() Config {
	return Config{
		States: map[string]Style{
			Working:   {Tab: "#3b82f6", Background: "#151b2b", Glyph: "🔵"},
			Attention: {Tab: "#f59e0b", Background: "#2a2112", Glyph: "🟡"},
			Done:      {Tab: "#22c55e", Background: "#13231a", Glyph: "🟢"},
			Error:     {Tab: "#ef4444", Background: "#2b1515", Glyph: "🔴"},
		},
		Statusline: Statusline{
			Max:     12,
			Sources: []string{"assistant", "tools", "user"},
			Label:   "hex",
			Prefix:  "",
			Empty:   "",
		},
	}
}

// Path returns the config file location.
func Path() string {
	if p := os.Getenv(EnvPath); p != "" {
		return p
	}
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "claude-colorizer", "config.json")
}

// Load returns defaults overlaid with the user's config. A missing file is
// not an error.
func Load() (Config, error) {
	cfg := Default()
	data, err := os.ReadFile(Path())
	if errors.Is(err, fs.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	var user Config
	if err := json.Unmarshal(data, &user); err != nil {
		return cfg, err
	}
	return merge(cfg, user), nil
}

func merge(base, user Config) Config {
	if user.Tab != nil {
		base.Tab = user.Tab
	}
	if user.Background != nil {
		base.Background = user.Background
	}
	if user.Title != nil {
		base.Title = user.Title
	}
	for name, u := range user.States {
		b := base.States[name]
		if u.Tab != "" {
			b.Tab = u.Tab
		}
		if u.Background != "" {
			b.Background = u.Background
		}
		if u.Glyph != "" {
			b.Glyph = u.Glyph
		}
		base.States[name] = b
	}
	s, u := &base.Statusline, user.Statusline
	if u.Max > 0 {
		s.Max = u.Max
	}
	if len(u.Sources) > 0 {
		s.Sources = u.Sources
	}
	if u.Label != "" {
		s.Label = u.Label
	}
	if u.Prefix != "" {
		s.Prefix = u.Prefix
	}
	if u.Empty != "" {
		s.Empty = u.Empty
	}
	return base
}
