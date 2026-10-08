package terminal

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cjodo/claude-colorizer/internal/colors"
)

func env(kv ...string) func(string) string {
	m := map[string]string{}
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i]] = kv[i+1]
	}
	return func(k string) string { return m[k] }
}

func TestDetect(t *testing.T) {
	cases := []struct {
		env  func(string) string
		want string
	}{
		{env("TERM", "xterm-kitty"), "kitty"},
		{env("KITTY_WINDOW_ID", "1", "TERM_PROGRAM", "tmux"), "kitty"},
		{env("TERM_PROGRAM", "ghostty"), "ghostty"},
		{env("TERM", "xterm-ghostty"), "ghostty"},
		{env("TERM_PROGRAM", "WezTerm"), "wezterm"},
		{env("WEZTERM_PANE", "3", "TERM_PROGRAM", "tmux"), "wezterm"},
		{env("TERM_PROGRAM", "iTerm.app"), "iterm2"},
		{env("LC_TERMINAL", "iTerm2"), "iterm2"},
		{env("TERM_PROGRAM", "WarpTerminal"), "warp"},
		{env("ALACRITTY_WINDOW_ID", "9"), "alacritty"},
		{env("WT_SESSION", "abc"), "windows-terminal"},
		{env("TERM", "xterm-256color"), "generic"},
		{env("TERM_PROGRAM", "ghostty", EnvOverride, "WezTerm"), "wezterm"},
	}
	for _, tc := range cases {
		if got := Detect(tc.env).Name(); got != tc.want {
			t.Errorf("Detect = %s, want %s", got, tc.want)
		}
	}
}

// Every driver must honor its declared capabilities.
func TestCapabilitiesMatchSequences(t *testing.T) {
	c := colors.Color{R: 0x3b, G: 0x82, B: 0xf6}
	for _, d := range All() {
		caps := d.Capabilities()
		if got := d.SetTabColor(c) != nil; got != caps.TabColor {
			t.Errorf("%s: SetTabColor non-nil=%v, TabColor cap=%v", d.Name(), got, caps.TabColor)
		}
		if got := d.SetBackground(c) != nil; got != caps.Background {
			t.Errorf("%s: SetBackground non-nil=%v, Background cap=%v", d.Name(), got, caps.Background)
		}
		if got := d.SetTitle("x") != nil; got != caps.Title {
			t.Errorf("%s: SetTitle non-nil=%v, Title cap=%v", d.Name(), got, caps.Title)
		}
	}
}

func TestSequences(t *testing.T) {
	c := colors.Color{R: 0x3b, G: 0x82, B: 0xf6}
	cases := []struct {
		name string
		got  []byte
		want string
	}{
		{"osc11", Ghostty{}.SetBackground(c), "\x1b]11;#3b82f6\x1b\\"},
		{"iterm tab", ITerm2{}.SetTabColor(c), "\x1b]6;1;bg;red;brightness;59\a\x1b]6;1;bg;green;brightness;130\a\x1b]6;1;bg;blue;brightness;246\a"},
		{"wezterm var", WezTerm{}.SetTabColor(c), "\x1b]1337;SetUserVar=claude_colorizer_tab=IzNiODJmNg==\a"},
		{"title sanitized", Alacritty{}.SetTitle("a\x1b]evil\ab"), "\x1b]2;a]evilb\x1b\\"},
	}
	for _, tc := range cases {
		if string(tc.got) != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, tc.got, tc.want)
		}
	}

	k := string(Kitty{}.SetTabColor(c))
	if !strings.HasPrefix(k, "\x1bP@kitty-cmd{") || !strings.Contains(k, `"active_bg":3900150`) || !strings.Contains(k, `"no_response":true`) {
		t.Errorf("kitty tab: %q", k)
	}
}

func TestTmuxWrap(t *testing.T) {
	var buf bytes.Buffer
	w := Writer{W: &buf, Tmux: true}
	// tmux applies background and title itself; other sequences pass through.
	if err := w.Write(nil, osc("11;#000000"), osc("111"), osc("1337;SetUserVar=x=eQ==")); err != nil {
		t.Fatal(err)
	}
	want := "\x1b]11;#000000\x1b\\" + "\x1b]111\x1b\\" +
		"\x1bPtmux;\x1b\x1b]1337;SetUserVar=x=eQ==\x1b\x1b\\\x1b\\"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}
