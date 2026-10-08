// Package transcript extracts colors from a Claude Code session transcript
// (JSONL, one message per line).
package transcript

import (
	"bytes"
	"encoding/json"
	"io"
	"os"

	"github.com/cjodo/claude-colorizer/internal/colors"
)

// Source selects which parts of the conversation are scanned.
type Source string

const (
	Assistant Source = "assistant" // Claude's prose
	Tools     Source = "tools"     // inputs Claude passed to tools (Edit/Write content, commands)
	User      Source = "user"      // your prompts
)

// tailBytes bounds how much of a long transcript is read per call; the
// statusline runs often and only recent colors matter.
const tailBytes = 512 << 10

type entry struct {
	Type    string `json:"type"`
	Message struct {
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

type block struct {
	Type  string          `json:"type"`
	Text  string          `json:"text"`
	Input json.RawMessage `json:"input"`
}

// RecentColors returns up to max distinct colors, most recent message first
// (in reading order within a message).
func RecentColors(path string, sources []Source, max int) ([]colors.Match, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := readTail(f, tailBytes)
	if err != nil {
		return nil, err
	}
	want := map[Source]bool{}
	for _, s := range sources {
		want[s] = true
	}

	lines := bytes.Split(data, []byte("\n"))
	seen := map[colors.Color]bool{}
	var out []colors.Match
	for i := len(lines) - 1; i >= 0 && len(out) < max; i-- {
		for _, text := range texts(lines[i], want) {
			for _, m := range colors.Find(text) {
				if seen[m.Color] {
					continue
				}
				seen[m.Color] = true
				out = append(out, m)
				if len(out) == max {
					return out, nil
				}
			}
		}
	}
	return out, nil
}

// texts returns the scannable strings in one transcript line.
func texts(line []byte, want map[Source]bool) []string {
	if len(bytes.TrimSpace(line)) == 0 {
		return nil
	}
	var e entry
	if json.Unmarshal(line, &e) != nil || len(e.Message.Content) == 0 {
		return nil
	}
	if e.Type != "user" && e.Type != "assistant" {
		return nil
	}

	// User prompts may be a bare string.
	var s string
	if json.Unmarshal(e.Message.Content, &s) == nil {
		if e.Type == "user" && want[User] {
			return []string{s}
		}
		return nil
	}

	var blocks []block
	if json.Unmarshal(e.Message.Content, &blocks) != nil {
		return nil
	}
	var out []string
	for _, b := range blocks {
		switch {
		case b.Type == "text" && e.Type == "assistant" && want[Assistant],
			b.Type == "text" && e.Type == "user" && want[User]:
			out = append(out, b.Text)
		case b.Type == "tool_use" && want[Tools]:
			out = appendStrings(out, b.Input)
		}
	}
	return out
}

// appendStrings collects every string value nested in a JSON document.
func appendStrings(out []string, raw json.RawMessage) []string {
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return out
	}
	var walk func(any)
	walk = func(v any) {
		switch t := v.(type) {
		case string:
			out = append(out, t)
		case []any:
			for _, x := range t {
				walk(x)
			}
		case map[string]any:
			for _, x := range t {
				walk(x)
			}
		}
	}
	walk(v)
	return out
}

// readTail returns the last n bytes of f, dropping a leading partial line.
func readTail(f *os.File, n int64) ([]byte, error) {
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	off := st.Size() - n
	if off <= 0 {
		return io.ReadAll(f)
	}
	buf := make([]byte, n)
	if _, err := f.ReadAt(buf, off); err != nil && err != io.EOF {
		return nil, err
	}
	if i := bytes.IndexByte(buf, '\n'); i >= 0 {
		buf = buf[i+1:]
	}
	return buf, nil
}
