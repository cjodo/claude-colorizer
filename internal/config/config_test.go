package config

import "testing"

func TestSwatchesEnabled(t *testing.T) {
	off, on := false, true
	cases := []struct {
		user *bool
		want bool
	}{
		{nil, true},
		{&off, false},
		{&on, true},
	}
	for _, tc := range cases {
		cfg := merge(Default(), Config{Statusline: Statusline{Swatches: tc.user}})
		if got := cfg.Statusline.SwatchesEnabled(); got != tc.want {
			t.Errorf("swatches=%v: SwatchesEnabled() = %v, want %v", tc.user, got, tc.want)
		}
	}
}

func TestTmuxBackgroundPassthrough(t *testing.T) {
	for user, want := range map[string]bool{"": false, "pane": false, "terminal": true} {
		cfg := merge(Default(), Config{TmuxBackground: user})
		if got := cfg.TmuxBackgroundPassthrough(); got != want {
			t.Errorf("tmuxBackground=%q: TmuxBackgroundPassthrough() = %v, want %v", user, got, want)
		}
	}
}
