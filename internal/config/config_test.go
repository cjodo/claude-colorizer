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
