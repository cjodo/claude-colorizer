package transcript

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sample = `{"type":"user","message":{"role":"user","content":"make it #ff0000-ish? try #00ff00"}}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Use #3b82f6 as primary and #ef4444 for errors."}]}}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Edit","input":{"file_path":"a.css","new_string":".btn { color: rgb(16, 185, 129); }"}}]}}
{"type":"user","message":{"role":"user","content":[{"type":"tool_result","content":"#abcdef ignored"}]}}
{"type":"summary","summary":"#123abc ignored"}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Final: #3b82f6 again"}]}}
`

func TestRecentColors(t *testing.T) {
	p := filepath.Join(t.TempDir(), "t.jsonl")
	if err := os.WriteFile(p, []byte(sample), 0o600); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		sources []Source
		max     int
		want    string
	}{
		{[]Source{Assistant, Tools, User}, 10, "#3b82f6 #10b981 #ef4444 #00ff00"},
		{[]Source{Assistant}, 10, "#3b82f6 #ef4444"},
		{[]Source{Tools}, 10, "#10b981"},
		{[]Source{Assistant, Tools, User}, 2, "#3b82f6 #10b981"},
	}
	for _, tc := range cases {
		got, err := RecentColors(p, tc.sources, tc.max)
		if err != nil {
			t.Fatal(err)
		}
		var hex []string
		for _, m := range got {
			hex = append(hex, m.Color.Hex())
		}
		if s := strings.Join(hex, " "); s != tc.want {
			t.Errorf("sources=%v max=%d: got %q, want %q", tc.sources, tc.max, s, tc.want)
		}
	}
}
