package colors

import "testing"

func TestFind(t *testing.T) {
	cases := []struct {
		in   string
		want []string // hex of each match, in order
	}{
		{"primary #3b82f6 and #FFF", []string{"#3b82f6", "#ffffff"}},
		{"with alpha #3b82f680 / #fff8", []string{"#3b82f6", "#ffffff"}},
		{"fixes #123 and #4567", nil},
		{"but #123456 is a color", []string{"#123456"}},
		{"see docs#section, &#123; and #abcdefg and #abc-def", nil},
		{"rgb(255, 0, 0) rgba(0,128,255,0.5)", []string{"#ff0000", "#0080ff"}},
		{"rgb(100% 0% 0% / 50%)", []string{"#ff0000"}},
		{"hsl(120, 100%, 50%) hsla(240deg 100% 50% / .3)", []string{"#00ff00", "#0000ff"}},
		{"oklch(62.8% 0.2577 29.23)", []string{"#ff0000"}},
		{"oklch(1 0 0)", []string{"#ffffff"}},
		{"mixed rgb(0,0,0) then #fff", []string{"#000000", "#ffffff"}},
	}
	for _, tc := range cases {
		got := Find(tc.in)
		if len(got) != len(tc.want) {
			t.Errorf("Find(%q) = %d matches %v, want %v", tc.in, len(got), got, tc.want)
			continue
		}
		for i, m := range got {
			if m.Color.Hex() != tc.want[i] {
				t.Errorf("Find(%q)[%d] = %s (%q), want %s", tc.in, i, m.Color.Hex(), m.Text, tc.want[i])
			}
		}
	}
}

func TestContrast(t *testing.T) {
	if (Color{255, 255, 0}).Contrast() != (Color{0, 0, 0}) {
		t.Error("yellow should get black text")
	}
	if (Color{0, 0, 128}).Contrast() != (Color{255, 255, 255}) {
		t.Error("navy should get white text")
	}
}
