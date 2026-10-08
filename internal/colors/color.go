// Package colors finds color literals in text (hex, rgb, hsl, oklch) and
// converts them to 24-bit sRGB, in the spirit of nvim-colorizer.
package colors

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Color is an opaque 24-bit sRGB color.
type Color struct{ R, G, B uint8 }

// Hex returns the color as #rrggbb.
func (c Color) Hex() string { return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B) }

// Luminance returns the WCAG relative luminance in [0,1].
func (c Color) Luminance() float64 {
	lin := func(v uint8) float64 {
		f := float64(v) / 255
		if f <= 0.04045 {
			return f / 12.92
		}
		return math.Pow((f+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(c.R) + 0.7152*lin(c.G) + 0.0722*lin(c.B)
}

// Contrast returns black or white, whichever reads better on c.
func (c Color) Contrast() Color {
	if c.Luminance() > 0.179 {
		return Color{0, 0, 0}
	}
	return Color{255, 255, 255}
}

// Mix blends c toward o by t in [0,1].
func (c Color) Mix(o Color, t float64) Color {
	m := func(a, b uint8) uint8 { return clamp8(float64(a) + (float64(b)-float64(a))*t) }
	return Color{m(c.R, o.R), m(c.G, o.G), m(c.B, o.B)}
}

// ParseHex parses #rgb, #rgba, #rrggbb or #rrggbbaa (leading # optional).
// Alpha is accepted and discarded.
func ParseHex(s string) (Color, error) {
	s = strings.TrimPrefix(s, "#")
	switch len(s) {
	case 3, 4:
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	case 6, 8:
		s = s[:6]
	default:
		return Color{}, fmt.Errorf("invalid hex color %q", s)
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return Color{}, fmt.Errorf("invalid hex color %q", s)
	}
	return Color{uint8(v >> 16), uint8(v >> 8), uint8(v)}, nil
}

func clamp8(f float64) uint8 {
	switch {
	case math.IsNaN(f) || f < 0:
		return 0
	case f > 255:
		return 255
	}
	return uint8(math.Round(f))
}

func hslToRGB(h, s, l float64) Color {
	h = math.Mod(math.Mod(h, 360)+360, 360) / 360
	if s == 0 {
		return Color{clamp8(l * 255), clamp8(l * 255), clamp8(l * 255)}
	}
	q := l * (1 + s)
	if l >= 0.5 {
		q = l + s - l*s
	}
	p := 2*l - q
	hue := func(t float64) float64 {
		if t < 0 {
			t++
		}
		if t > 1 {
			t--
		}
		switch {
		case t < 1.0/6:
			return p + (q-p)*6*t
		case t < 1.0/2:
			return q
		case t < 2.0/3:
			return p + (q-p)*(2.0/3-t)*6
		}
		return p
	}
	return Color{clamp8(hue(h+1.0/3) * 255), clamp8(hue(h) * 255), clamp8(hue(h-1.0/3) * 255)}
}

// oklchToRGB converts OKLCH (L in [0,1], C, H in degrees) to gamut-clipped sRGB.
func oklchToRGB(l, c, h float64) Color {
	hr := h * math.Pi / 180
	a, b := c*math.Cos(hr), c*math.Sin(hr)

	l_ := l + 0.3963377774*a + 0.2158037573*b
	m_ := l - 0.1055613458*a - 0.0638541728*b
	s_ := l - 0.0894841775*a - 1.2914855480*b
	l3, m3, s3 := l_*l_*l_, m_*m_*m_, s_*s_*s_

	r := 4.0767416621*l3 - 3.3077115913*m3 + 0.2309699292*s3
	g := -1.2684380046*l3 + 2.6097574011*m3 - 0.3413193965*s3
	bl := -0.0041960863*l3 - 0.7034186147*m3 + 1.7076147010*s3

	gamma := func(x float64) uint8 {
		if x <= 0.0031308 {
			return clamp8(12.92 * x * 255)
		}
		return clamp8((1.055*math.Pow(x, 1/2.4) - 0.055) * 255)
	}
	return Color{gamma(r), gamma(g), gamma(bl)}
}
