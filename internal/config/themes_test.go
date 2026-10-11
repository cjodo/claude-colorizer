package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cjodo/claude-colorizer/internal/colors"
)

var allStates = []string{Idle, Working, Attention, Done, Error}

func TestThemesValid(t *testing.T) {
	def := Default().States
	for _, name := range ThemeNames() {
		states, err := ThemeStates(name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for _, s := range allStates {
			st := states[s]
			for _, h := range []string{st.Tab, st.Background} {
				if _, err := colors.ParseHex(h); err != nil || len(h) != 7 {
					t.Errorf("%s/%s: bad color %q", name, s, h)
				}
			}
			if st.Glyph != def[s].Glyph {
				t.Errorf("%s/%s: glyph changed to %q", name, s, st.Glyph)
			}
			if name == DefaultTheme {
				if st.Foreground != "" {
					t.Errorf("default/%s: foreground %q, want the terminal's own", s, st.Foreground)
				}
				continue
			}
			fg, err := colors.ParseHex(st.Foreground)
			if err != nil || len(st.Foreground) != 7 {
				t.Errorf("%s/%s: bad foreground %q", name, s, st.Foreground)
				continue
			}
			bg, _ := colors.ParseHex(st.Background)
			if r := contrast(fg, bg); r < 4.5 {
				t.Errorf("%s/%s: text %s on %s has contrast %.2f, want >= 4.5", name, s, st.Foreground, st.Background, r)
			}
		}
	}
}

// contrast is the WCAG contrast ratio between two colors.
func contrast(a, b colors.Color) float64 {
	la, lb := a.Luminance()+0.05, b.Luminance()+0.05
	if la < lb {
		la, lb = lb, la
	}
	return la / lb
}

func TestResolveTheme(t *testing.T) {
	for in, want := range map[string]string{
		"":           DefaultTheme,
		"Dracula":    "dracula",
		"catppuccin": "catppuccin-mocha",
		" GITHUB ":   "github-dark",
		"gruvbox":    "gruvbox-dark",
		"rose-pine":  "rose-pine",
		"default":    DefaultTheme,
	} {
		got, err := ResolveTheme(in)
		if err != nil || got != want {
			t.Errorf("ResolveTheme(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := ResolveTheme("nope"); err == nil {
		t.Error("ResolveTheme(nope): want error")
	}
}

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "claude-colorizer", "config.json")
	t.Setenv(EnvPath, path)
	if body != "" {
		os.MkdirAll(filepath.Dir(path), 0o755)
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func TestLoadTheme(t *testing.T) {
	writeConfig(t, `{"theme": "dracula", "states": {"working": {"tab": "#123abc"}}}`)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	want, _ := ThemeStates("dracula")
	if got := cfg.States[Working]; got.Tab != "#123abc" || got.Background != want[Working].Background {
		t.Errorf("working = %+v; want user tab over theme background %s", got, want[Working].Background)
	}
	if got := cfg.States[Done]; got != want[Done] {
		t.Errorf("done = %+v, want %+v", got, want[Done])
	}
}

func TestLoadThemeForeground(t *testing.T) {
	writeConfig(t, `{"theme": "catppuccin-latte", "states": {"done": {"foreground": "#000000"}}}`)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.States[Working].Foreground; got != "#4c4f69" {
		t.Errorf("working foreground = %q, want the theme's #4c4f69", got)
	}
	if got := cfg.States[Done].Foreground; got != "#000000" {
		t.Errorf("done foreground = %q, want user's #000000", got)
	}
}

func TestLoadUnknownTheme(t *testing.T) {
	writeConfig(t, `{"theme": "nope", "title": false}`)
	cfg, err := Load()
	if err == nil {
		t.Fatal("want error for unknown theme")
	}
	if cfg.States[Working] != Default().States[Working] {
		t.Errorf("working = %+v, want defaults", cfg.States[Working])
	}
	if cfg.TitleEnabled() {
		t.Error("other keys should still apply")
	}
}

func TestSetTheme(t *testing.T) {
	path := writeConfig(t, "")
	if err := SetTheme("Tokyo-Night"); err != nil {
		t.Fatal(err)
	}
	if cfg, _ := Load(); cfg.Theme != "tokyo-night" {
		t.Errorf("theme = %q after creating file", cfg.Theme)
	}

	os.WriteFile(path, []byte(`{"statusline": {"max": 3}, "theme": "dracula"}`), 0o644)
	if err := SetTheme("catppuccin"); err != nil {
		t.Fatal(err)
	}
	var raw map[string]json.RawMessage
	data, _ := os.ReadFile(path)
	json.Unmarshal(data, &raw)
	if string(raw["theme"]) != `"catppuccin-mocha"` || !strings.Contains(string(raw["statusline"]), `"max": 3`) {
		t.Errorf("config after SetTheme:\n%s", data)
	}

	if err := SetTheme("nope"); err == nil {
		t.Error("SetTheme(nope): want error")
	}
	if after, _ := os.ReadFile(path); string(after) != string(data) {
		t.Error("unknown theme modified the file")
	}

	if err := SetTheme("default"); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(path)
	if strings.Contains(string(data), "theme") {
		t.Errorf("default should remove the key:\n%s", data)
	}
}
