package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cjodo/claude-colorizer/internal/colors"
)

// DefaultTheme names the built-in colors from Default().
const DefaultTheme = "default"

// A palette holds a theme's editor background and text color, and one
// accent per state. Each state's tab is its accent; its background tint is
// Base mixed slightly toward the accent. Fg is set as the terminal's text
// color so the tint stays readable whatever the terminal's own theme is.
type palette struct {
	Base, Fg, Idle, Working, Attention, Done, Error string
}

// tintAmount is how far a state's background moves from Base toward its
// accent.
const tintAmount = 0.12

var themes = map[string]palette{
	"dracula":              {"#282a36", "#f8f8f2", "#6272a4", "#bd93f9", "#ffb86c", "#50fa7b", "#ff5555"},
	"one-dark-pro":         {"#282c34", "#abb2bf", "#5c6370", "#61afef", "#e5c07b", "#98c379", "#e06c75"},
	"tokyo-night":          {"#1a1b26", "#c0caf5", "#565f89", "#7aa2f7", "#e0af68", "#9ece6a", "#f7768e"},
	"catppuccin-mocha":     {"#1e1e2e", "#cdd6f4", "#7f849c", "#89b4fa", "#f9e2af", "#a6e3a1", "#f38ba8"},
	"catppuccin-macchiato": {"#24273a", "#cad3f5", "#8087a2", "#8aadf4", "#eed49f", "#a6da95", "#ed8796"},
	"catppuccin-frappe":    {"#303446", "#c6d0f5", "#838ba7", "#8caaee", "#e5c890", "#a6d189", "#e78284"},
	"catppuccin-latte":     {"#eff1f5", "#4c4f69", "#8c8fa1", "#1e66f5", "#df8e1d", "#40a02b", "#d20f39"},
	"gruvbox-dark":         {"#282828", "#ebdbb2", "#928374", "#83a598", "#fabd2f", "#b8bb26", "#fb4934"},
	"gruvbox-light":        {"#fbf1c7", "#3c3836", "#928374", "#076678", "#b57614", "#79740e", "#9d0006"},
	"github-dark":          {"#0d1117", "#e6edf3", "#8b949e", "#58a6ff", "#d29922", "#3fb950", "#f85149"},
	"github-light":         {"#ffffff", "#1f2328", "#656d76", "#0969da", "#9a6700", "#1a7f37", "#d1242f"},
	"night-owl":            {"#011627", "#d6deeb", "#637777", "#82aaff", "#ecc48d", "#addb67", "#ef5350"},
	// Cobalt2's comment color is the same blue as its accent, so idle uses
	// a muted slate instead.
	"cobalt2": {"#193549", "#ffffff", "#8f9fb0", "#0088ff", "#ffc600", "#3ad900", "#ff628c"},
	// Rosé Pine has no green: done uses foam.
	"rose-pine":      {"#191724", "#e0def4", "#6e6a86", "#c4a7e7", "#f6c177", "#9ccfd8", "#eb6f92"},
	"rose-pine-moon": {"#232136", "#e0def4", "#6e6a86", "#c4a7e7", "#f6c177", "#9ccfd8", "#eb6f92"},
	"rose-pine-dawn": {"#faf4ed", "#575279", "#9893a5", "#907aa9", "#ea9d34", "#56949f", "#b4637a"},
}

var themeAliases = map[string]string{
	"catppuccin": "catppuccin-mocha",
	"gruvbox":    "gruvbox-dark",
	"github":     "github-dark",
}

// ThemeNames returns the built-in theme names, sorted, with "default" first.
func ThemeNames() []string {
	names := make([]string, 0, len(themes))
	for n := range themes {
		names = append(names, n)
	}
	sort.Strings(names)
	return append([]string{DefaultTheme}, names...)
}

// ResolveTheme returns the canonical name for a theme or alias, matched
// case-insensitively.
func ResolveTheme(name string) (string, error) {
	n := strings.ToLower(strings.TrimSpace(name))
	if a, ok := themeAliases[n]; ok {
		n = a
	}
	if n == "" {
		return DefaultTheme, nil
	}
	if _, ok := themes[n]; ok || n == DefaultTheme {
		return n, nil
	}
	return "", fmt.Errorf("unknown theme %q; run `claude-colorizer theme` to list them", name)
}

// ThemeStates returns the state colors for a theme, with the default glyphs.
func ThemeStates(name string) (map[string]Style, error) {
	states := Default().States
	return states, applyTheme(states, name)
}

// applyTheme sets the tab, background and text color of each state from
// the named theme. Glyphs are left alone.
func applyTheme(states map[string]Style, name string) error {
	n, err := ResolveTheme(name)
	if err != nil || n == DefaultTheme {
		return err
	}
	p := themes[n]
	base, err := colors.ParseHex(p.Base)
	if err != nil {
		return err
	}
	fg, err := colors.ParseHex(p.Fg)
	if err != nil {
		return err
	}
	for state, accent := range map[string]string{
		Idle: p.Idle, Working: p.Working, Attention: p.Attention, Done: p.Done, Error: p.Error,
	} {
		c, err := colors.ParseHex(accent)
		if err != nil {
			return err
		}
		s := states[state]
		s.Tab = c.Hex()
		s.Background = base.Mix(c, tintAmount).Hex()
		s.Foreground = fg.Hex()
		states[state] = s
	}
	return nil
}

// SetTheme writes "theme" into the config file, keeping every other key.
// "default" removes it. The file and its directory are created if missing.
func SetTheme(name string) error {
	n, err := ResolveTheme(name)
	if err != nil {
		return err
	}
	path := Path()
	raw := map[string]json.RawMessage{}
	data, err := os.ReadFile(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return err
	case len(strings.TrimSpace(string(data))) > 0:
		if err := json.Unmarshal(data, &raw); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
	}
	if n == DefaultTheme {
		delete(raw, "theme")
	} else {
		raw["theme"], _ = json.Marshal(n)
	}
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0o644)
}
