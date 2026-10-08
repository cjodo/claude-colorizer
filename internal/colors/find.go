package colors

import (
	"cmp"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Match is a color literal found in text. Start and End are byte offsets.
type Match struct {
	Start, End int
	Text       string
	Color      Color
}

var (
	hexRe = regexp.MustCompile(`#(?:[0-9a-fA-F]{8}|[0-9a-fA-F]{6}|[0-9a-fA-F]{3,4})`)
	// rgb()/rgba()/hsl()/hsla()/oklch() with comma or space syntax and an
	// optional alpha after "," or "/".
	fnRe = regexp.MustCompile(`(?i)\b(rgba?|hsla?|oklch)\(\s*([-+.\d]+(?:%|deg)?)\s*[,\s]\s*([-+.\d]+%?)\s*[,\s]\s*([-+.\d]+(?:%|deg)?)\s*(?:[,/]\s*[-+.\d]+%?\s*)?\)`)
)

// Find returns every color literal in s, in order of appearance.
func Find(s string) []Match {
	var out []Match
	for _, loc := range hexRe.FindAllStringIndex(s, -1) {
		if !hexBoundary(s, loc[0], loc[1]) {
			continue
		}
		text := s[loc[0]:loc[1]]
		// "#123" is far more often an issue/PR number than a color, so short
		// forms must contain at least one hex letter.
		if len(text) <= 5 && !strings.ContainsAny(text[1:], "abcdefABCDEF") {
			continue
		}
		c, err := ParseHex(text)
		if err != nil {
			continue
		}
		out = append(out, Match{loc[0], loc[1], text, c})
	}
	for _, m := range fnRe.FindAllStringSubmatchIndex(s, -1) {
		args := []string{s[m[4]:m[5]], s[m[6]:m[7]], s[m[8]:m[9]]}
		c, ok := parseFunc(strings.ToLower(s[m[2]:m[3]]), args)
		if !ok {
			continue
		}
		out = append(out, Match{m[0], m[1], s[m[0]:m[1]], c})
	}
	slices.SortFunc(out, func(a, b Match) int { return cmp.Compare(a.Start, b.Start) })
	return out
}

// hexBoundary rejects hex runs embedded in longer tokens such as URL
// fragments (#section), HTML entities (&#123;) or longer hex strings.
func hexBoundary(s string, start, end int) bool {
	if start > 0 {
		p := s[start-1]
		if p == '&' || isWord(p) {
			return false
		}
	}
	if end < len(s) && (isWord(s[end]) || s[end] == '-') {
		return false
	}
	return true
}

func isWord(b byte) bool {
	return b == '_' || b >= '0' && b <= '9' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z'
}

func parseFunc(name string, a []string) (Color, bool) {
	switch name {
	case "rgb", "rgba":
		var ch [3]uint8
		for i, v := range a {
			f, pct, ok := num(v)
			if !ok {
				return Color{}, false
			}
			if pct {
				f = f * 255 / 100
			}
			ch[i] = clamp8(f)
		}
		return Color{ch[0], ch[1], ch[2]}, true
	case "hsl", "hsla":
		h, _, ok1 := num(a[0])
		sat, _, ok2 := num(a[1])
		l, _, ok3 := num(a[2])
		if !ok1 || !ok2 || !ok3 {
			return Color{}, false
		}
		return hslToRGB(h, sat/100, l/100), true
	case "oklch":
		l, pct, ok1 := num(a[0])
		c, cpct, ok2 := num(a[1])
		h, _, ok3 := num(a[2])
		if !ok1 || !ok2 || !ok3 {
			return Color{}, false
		}
		if pct || l > 1 {
			l /= 100
		}
		if cpct {
			c = c * 0.4 / 100
		}
		return oklchToRGB(l, c, h), true
	}
	return Color{}, false
}

// num parses "12", "12.5%", "120deg"; pct reports a trailing %.
func num(s string) (f float64, pct bool, ok bool) {
	s = strings.TrimSuffix(s, "deg")
	if strings.HasSuffix(s, "%") {
		pct, s = true, s[:len(s)-1]
	}
	f, err := strconv.ParseFloat(s, 64)
	return f, pct, err == nil
}
