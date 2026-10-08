package terminal

import (
	"encoding/base64"
	"unicode/utf8"
)

// Program Status Protocol (OSC 7501): the program reports what it is doing
// and the terminal decides how to present it (tab badge, notification, ...).
// It is terminal-agnostic and terminals ignore unknown OSCs, so it is sent
// to every terminal without detecting support first.
//
// Spec: https://www.superlogical.com/rex/docs/build/program-status

// Program status states.
const (
	StatusIdle    = "idle"
	StatusWorking = "working"
	StatusDone    = "done"
	StatusBlocked = "blocked"
	StatusError   = "error"
	StatusClear   = "clear"
)

// Status is one OSC 7501 report for the root record. Empty fields are omitted.
type Status struct {
	State string
	Kind  string // permission, question or auth; only with blocked
	App   string
	Title string
	Msg   string
}

// Spec limits on decoded text; longer reports are discarded whole.
const (
	statusTitleMax = 192
	statusMsgMax   = 2048
)

// ProgramStatus builds the OSC 7501 report for s.
func ProgramStatus(s Status) []byte {
	body := "state=" + s.State
	if s.State == StatusClear {
		return osc("7501;" + body)
	}
	if s.Kind != "" && s.State == StatusBlocked {
		body += ":kind=" + s.Kind
	}
	if s.App != "" {
		body += ":app=" + s.App
	}
	if t := statusText(s.Title, statusTitleMax); t != "" {
		body += ":title=" + t
	}
	if m := statusText(s.Msg, statusMsgMax); m != "" {
		body += ":msg=" + m
	}
	return osc("7501;" + body)
}

// statusText sanitizes, truncates on a UTF-8 boundary and base64-encodes s.
func statusText(s string, max int) string {
	s = sanitizeStatus(s)
	if len(s) > max {
		s = s[:max]
		for !utf8.ValidString(s) {
			s = s[:len(s)-1]
		}
	}
	if s == "" {
		return ""
	}
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// sanitizeStatus replaces C0, DEL and C1 controls (forbidden in decoded
// text) with spaces, so multi-line messages become one line.
func sanitizeStatus(s string) string {
	r := []rune(s)
	for i, c := range r {
		if c < 0x20 || (c >= 0x7f && c <= 0x9f) {
			r[i] = ' '
		}
	}
	return string(r)
}
