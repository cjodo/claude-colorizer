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

// Inside tmux, the attached client ttys win over stale environment variables.
func TestDetectTmuxClients(t *testing.T) {
	defer func(f func(string) []string) { tmuxClients = f }(tmuxClients)
	stale := env("TMUX", "/tmp/tmux-1000/default,1,5", "TMUX_PANE", "%15",
		"TERM_PROGRAM", "tmux", "GHOSTTY_RESOURCES_DIR", "/usr/share/ghostty")

	cases := []struct {
		clients []string
		want    string
	}{
		{[]string{"kitty(0.49.2)|xterm-kitty"}, "kitty"},
		{[]string{"kitty(0.49.2)|xterm-kitty", "kitty(0.49.2)|xterm-kitty"}, "kitty"},
		{[]string{"kitty(0.49.2)|xterm-kitty", "ghostty_1.3.1|xterm-ghostty"}, "kitty+ghostty"},
		{[]string{"|screen-256color"}, "generic"},
		{nil, "ghostty"}, // no clients (e.g. tmux unreachable): fall back to env
	}
	for _, tc := range cases {
		var gotPane string
		tmuxClients = func(pane string) []string { gotPane = pane; return tc.clients }
		if got := Detect(stale).Name(); got != tc.want {
			t.Errorf("clients %q: Detect = %s, want %s", tc.clients, got, tc.want)
		}
		if gotPane != "%15" {
			t.Errorf("tmuxClients pane = %q, want %%15", gotPane)
		}
	}
}

func TestMulti(t *testing.T) {
	m := Multi{Ghostty{}, Kitty{}}
	c := colors.Color{R: 1, G: 2, B: 3}
	if !m.Capabilities().TabColor {
		t.Error("Multi with kitty should report tab color")
	}
	if got, want := string(m.SetTabColor(c)), string(Kitty{}.SetTabColor(c)); got != want {
		t.Errorf("SetTabColor = %q, want kitty's %q", got, want)
	}
	if got, want := string(m.SetBackground(c)), string(Ghostty{}.SetBackground(c)); got != want {
		t.Errorf("SetBackground = %q, want a single OSC 11 %q", got, want)
	}
}

func TestProgramStatus(t *testing.T) {
	cases := []struct {
		name string
		s    Status
		want string
	}{
		{"clear drops other keys", Status{State: StatusClear, App: "x", Title: "t"}, "\x1b]7501;state=clear\x1b\\"},
		{"working", Status{State: StatusWorking, App: "claude-code", Title: "Claude Code"},
			"\x1b]7501;state=working:app=claude-code:title=Q2xhdWRlIENvZGU=\x1b\\"},
		{"blocked kind+msg", Status{State: StatusBlocked, Kind: "permission", Msg: "ok?"},
			"\x1b]7501;state=blocked:kind=permission:msg=b2s/\x1b\\"},
		{"kind only with blocked", Status{State: StatusDone, Kind: "permission"}, "\x1b]7501;state=done\x1b\\"},
		{"msg made one line", Status{State: StatusError, Msg: "a\nb"}, "\x1b]7501;state=error:msg=YSBi\x1b\\"},
	}
	for _, tc := range cases {
		if got := string(ProgramStatus(tc.s)); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}

	// Over-long text is truncated on a rune boundary, not dropped.
	long := strings.Repeat("é", statusTitleMax) // 2 bytes per rune
	if got := string(ProgramStatus(Status{State: StatusIdle, Title: long})); !strings.Contains(got, ":title=") || len(got) > 4096 {
		t.Errorf("long title: %q", got)
	}
	if got := statusText(long, 5); got != "w6nDqQ==" { // "éé"
		t.Errorf("truncate: got %q", got)
	}
}

// OSC 7501 isn't understood by tmux, so it must pass through to the outer terminal.
func TestProgramStatusTmux(t *testing.T) {
	var buf bytes.Buffer
	if err := (Writer{W: &buf, Tmux: true}).Write(ProgramStatus(Status{State: StatusDone})); err != nil {
		t.Fatal(err)
	}
	if want := "\x1bPtmux;\x1b\x1b]7501;state=done\x1b\x1b\\\x1b\\"; buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}
