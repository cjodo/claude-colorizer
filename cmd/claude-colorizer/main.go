// Command claude-colorizer visualizes colors for Claude Code sessions:
//
//   - statusline: swatches for colors mentioned in the conversation
//   - hook:       recolors the terminal tab/background by session state
//   - show:       highlights color literals in text, like nvim-colorizer
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cjodo/claude-colorizer/internal/colors"
	"github.com/cjodo/claude-colorizer/internal/config"
	"github.com/cjodo/claude-colorizer/internal/session"
	"github.com/cjodo/claude-colorizer/internal/setup"
	"github.com/cjodo/claude-colorizer/internal/terminal"
	"github.com/cjodo/claude-colorizer/internal/transcript"
)

const usage = `claude-colorizer — color visualization for Claude Code

Usage:
  claude-colorizer statusline [-- CMD...]  Claude Code statusLine: swatches of recent colors.
                                           With CMD, its output is shown on the line above.
  claude-colorizer hook                    Claude Code hook handler (reads event JSON on stdin).
  claude-colorizer show [FILE...]          Highlight color literals in files or stdin.
  claude-colorizer set STATE               Apply a state: idle|working|attention|done|error.
  claude-colorizer try COLOR               Set the terminal background to COLOR to preview it.
  claude-colorizer reset                   Restore tab color, background and title.
  claude-colorizer detect                  Show the detected terminal and its capabilities.
  claude-colorizer config                  Print the effective configuration and its path.
  claude-colorizer install [FLAGS]         Add the statusline and hooks to ~/.claude/settings.json.
  claude-colorizer uninstall [FLAGS]       Remove them again (restores a chained statusline).
      --no-hooks  --no-statusline  --dry-run  --settings PATH

Environment:
  CLAUDE_COLORIZER_TERMINAL  force a driver: kitty, ghostty, wezterm, iterm2, warp,
                             alacritty, windows-terminal, generic
  CLAUDE_COLORIZER_CONFIG    config file path (default ~/.config/claude-colorizer/config.json)
  CLAUDE_COLORIZER_STATE_DIR per-session state for the statusline indicator
                             (default: user cache dir/claude-colorizer/sessions)
  CLAUDE_COLORIZER_DEBUG     print hook errors to stderr
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	cfg, cfgErr := config.Load()

	var err error
	switch cmd, args := os.Args[1], os.Args[2:]; cmd {
	case "statusline":
		err = runStatusline(cfg, args)
	case "hook":
		// Hooks must never break the session: report only when debugging.
		if err := runHook(cfg, os.Stdin); err != nil && os.Getenv("CLAUDE_COLORIZER_DEBUG") != "" {
			fmt.Fprintln(os.Stderr, "claude-colorizer:", err)
		}
		return
	case "show":
		err = runShow(args)
	case "set":
		if len(args) != 1 {
			err = fmt.Errorf("usage: set STATE")
			break
		}
		cwd, _ := os.Getwd()
		err = applyState(cfg, args[0], hookInput{Cwd: cwd})
	case "try":
		err = runTry(args)
	case "reset":
		err = applyState(cfg, "", hookInput{})
	case "detect":
		runDetect(cfg)
	case "install", "uninstall":
		err = runSetup(cmd, args)
	case "config":
		if cfgErr != nil {
			fmt.Fprintln(os.Stderr, "warning:", cfgErr)
		}
		fmt.Println("#", config.Path())
		b, _ := json.MarshalIndent(cfg, "", "  ")
		fmt.Println(string(b))
	case "-h", "--help", "help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", cmd, usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "claude-colorizer:", err)
		os.Exit(1)
	}
}

// ---------------------------------------------------------------------------
// statusline

type statusInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
}

func runStatusline(cfg config.Config, args []string) error {
	in, err := io.ReadAll(os.Stdin)
	if err != nil {
		return err
	}

	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	if len(args) > 0 {
		// Chain the user's existing statusline: same stdin, its output first.
		c := exec.Command(args[0], args[1:]...)
		c.Stdin = bytes.NewReader(in)
		c.Stderr = os.Stderr
		if out, err := c.Output(); err == nil && len(bytes.TrimSpace(out)) > 0 {
			fmt.Println(strings.TrimRight(string(out), "\n"))
		}
	}

	var si statusInput
	_ = json.Unmarshal(in, &si)
	var matches []colors.Match
	if si.TranscriptPath != "" {
		var sources []transcript.Source
		for _, s := range cfg.Statusline.Sources {
			sources = append(sources, transcript.Source(s))
		}
		matches, _ = transcript.RecentColors(si.TranscriptPath, sources, cfg.Statusline.Max)
	}
	line := renderSwatches(cfg.Statusline, matches)
	if ind := renderIndicator(cfg, session.Load(si.SessionID)); ind != "" {
		if line != "" {
			ind += "  "
		}
		line = ind + line
	}
	fmt.Println(line)
	return nil
}

// renderIndicator shows the session's current state as a dot in its tab
// color, optionally followed by the state name.
func renderIndicator(cfg config.Config, state string) string {
	mode := cfg.Statusline.Indicator
	style, ok := cfg.States[state]
	c, err := colors.ParseHex(style.Tab)
	if !ok || err != nil || mode == "none" {
		return ""
	}
	s := fg(c) + "●" + sgrReset
	if mode != "dot" {
		s += " " + state
	}
	return s
}

func renderSwatches(sc config.Statusline, matches []colors.Match) string {
	if len(matches) == 0 {
		return sc.Empty
	}
	var b strings.Builder
	b.WriteString(sc.Prefix)
	for i, m := range matches {
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(bg(m.Color) + "  " + sgrReset)
		switch sc.Label {
		case "none":
		case "original":
			b.WriteString(" " + m.Text)
		default:
			b.WriteString(" " + m.Color.Hex())
		}
	}
	return b.String()
}

// ---------------------------------------------------------------------------
// hook

type hookInput struct {
	Event            string `json:"hook_event_name"`
	SessionID        string `json:"session_id"`
	Cwd              string `json:"cwd"`
	Message          string `json:"message"`           // Notification
	NotificationType string `json:"notification_type"` // Notification
}

// stateFor maps a Claude Code hook event to a state; "" means reset, ok=false
// means the event is ignored.
func stateFor(event string) (state string, ok bool) {
	switch event {
	case "SessionStart":
		return config.Idle, true
	case "SessionEnd":
		return "", true
	case "UserPromptSubmit", "PostToolUse":
		return config.Working, true
	case "Notification":
		return config.Attention, true
	case "Stop":
		return config.Done, true
	case "StopFailure", "PostToolUseFailure":
		return config.Error, true
	}
	return "", false
}

func runHook(cfg config.Config, r io.Reader) error {
	var in hookInput
	if err := json.NewDecoder(r).Decode(&in); err != nil {
		return err
	}
	state, ok := stateFor(in.Event)
	if !ok {
		return nil
	}
	// Record it for the statusline indicator even if the terminal is unreachable.
	saveErr := session.Save(in.SessionID, state)
	if err := applyState(cfg, state, in); err != nil {
		return err
	}
	return saveErr
}

// applyState recolors the terminal for state, or restores it when state is "".
func applyState(cfg config.Config, state string, in hookInput) error {
	term := terminal.Detect(os.Getenv)
	caps := term.Capabilities()
	cwd := in.Cwd

	var seqs [][]byte
	if ps := programStatus(state, in); cfg.StatusEnabled() && ps.State != "" {
		seqs = append(seqs, terminal.ProgramStatus(ps))
	}
	if state == "" {
		if cfg.TabEnabled() {
			seqs = append(seqs, term.ResetTabColor())
		}
		if cfg.BackgroundEnabled() {
			seqs = append(seqs, term.ResetBackground())
		}
		if cfg.TitleEnabled() && !caps.TabColor {
			seqs = append(seqs, term.SetTitle(title("", cwd)))
		}
	} else {
		style, ok := cfg.States[state]
		if !ok {
			return fmt.Errorf("unknown state %q", state)
		}
		if c, err := colors.ParseHex(style.Tab); err == nil && cfg.TabEnabled() {
			seqs = append(seqs, term.SetTabColor(c))
		}
		if c, err := colors.ParseHex(style.Background); err == nil && cfg.BackgroundEnabled() {
			seqs = append(seqs, term.SetBackground(c))
		}
		// Terminals without colorable tabs get a glyph in the title instead.
		if cfg.TitleEnabled() && (!caps.TabColor || !cfg.TabEnabled()) {
			seqs = append(seqs, term.SetTitle(title(style.Glyph, cwd)))
		}
	}
	return writeTTY(seqs...)
}

// programStatus maps a state to an OSC 7501 report, so terminals that speak
// the Program Status Protocol can show the session state natively. Custom
// states have no protocol equivalent and get an empty State.
func programStatus(state string, in hookInput) terminal.Status {
	s := terminal.Status{App: "claude-code", Title: title("", in.Cwd)}
	switch state {
	case "":
		s = terminal.Status{State: terminal.StatusClear}
	case config.Idle:
		s.State = terminal.StatusIdle
	case config.Working:
		s.State = terminal.StatusWorking
	case config.Attention:
		s.State = terminal.StatusBlocked
		s.Msg = in.Message
		switch in.NotificationType {
		case "permission_prompt":
			s.Kind = "permission"
		case "elicitation_dialog":
			s.Kind = "question"
		}
	case config.Done:
		s.State = terminal.StatusDone
	case config.Error:
		s.State = terminal.StatusError
	}
	return s
}

func title(glyph, cwd string) string {
	t := "Claude Code"
	if cwd != "" {
		t += " · " + filepath.Base(cwd)
	}
	if glyph != "" {
		t = glyph + " " + t
	}
	return t
}

func writeTTY(seqs ...[]byte) error {
	tty, err := terminal.OpenTTY()
	if err != nil {
		return err
	}
	defer tty.Close()
	return terminal.Writer{W: tty, Tmux: terminal.InTmux(os.Getenv)}.Write(seqs...)
}

// ---------------------------------------------------------------------------
// install / uninstall

func runSetup(cmd string, args []string) error {
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	noHooks := fs.Bool("no-hooks", false, "leave hooks alone (e.g. when using the plugin)")
	noStatus := fs.Bool("no-statusline", false, "leave statusLine alone")
	dryRun := fs.Bool("dry-run", false, "print the changes without writing")
	settings := fs.String("settings", setup.DefaultSettingsPath(), "Claude Code settings.json to edit")
	if err := fs.Parse(args); err != nil {
		return err
	}

	bin, err := os.Executable()
	if err != nil {
		return err
	}
	if bin, err = filepath.EvalSymlinks(bin); err != nil {
		return err
	}
	if strings.Contains(bin, "go-build") {
		return fmt.Errorf("refusing to install a temporary `go run` binary; use `go install` or `make install`")
	}

	o := setup.Options{
		Settings:   *settings,
		Binary:     bin,
		StateDir:   filepath.Dir(config.Path()),
		Hooks:      !*noHooks,
		Statusline: !*noStatus,
		DryRun:     *dryRun,
	}
	var changes []string
	if cmd == "install" {
		changes, err = setup.Install(o)
	} else {
		changes, err = setup.Uninstall(o)
	}
	if err != nil {
		return err
	}
	prefix := ""
	if *dryRun {
		prefix = "(dry run) "
	}
	fmt.Printf("%s%s\n", prefix, *settings)
	for _, c := range changes {
		fmt.Printf("  %s\n", c)
	}
	if cmd == "install" && !*dryRun {
		fmt.Printf("  binary: %s\nRestart Claude Code to pick up the changes. Terminal: ", bin)
		runDetectName()
	}
	return nil
}

// ---------------------------------------------------------------------------
// show / try / detect

func runShow(files []string) error {
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()
	if len(files) == 0 {
		return highlight(out, os.Stdin)
	}
	for _, f := range files {
		r, err := os.Open(f)
		if err != nil {
			return err
		}
		err = highlight(out, r)
		r.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func highlight(w io.Writer, r io.Reader) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64<<10), 16<<20)
	for sc.Scan() {
		line := sc.Text()
		last := 0
		for _, m := range colors.Find(line) {
			if m.Start < last {
				continue
			}
			fmt.Fprint(w, line[last:m.Start], bg(m.Color), fg(m.Color.Contrast()), m.Text, sgrReset)
			last = m.End
		}
		fmt.Fprintln(w, line[last:])
	}
	return sc.Err()
}

func runTry(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: try COLOR (e.g. '#1e1e2e' or 'oklch(25%% 0.03 270)')")
	}
	ms := colors.Find(args[0])
	if len(ms) == 0 {
		c, err := colors.ParseHex(args[0])
		if err != nil {
			return err
		}
		ms = []colors.Match{{Color: c}}
	}
	term := terminal.Detect(os.Getenv)
	seq := term.SetBackground(ms[0].Color)
	if seq == nil {
		return fmt.Errorf("%s cannot set the background color", term.Name())
	}
	fmt.Printf("background → %s  (run `claude-colorizer reset` to restore)\n", ms[0].Color.Hex())
	return writeTTY(seq)
}

func runDetectName() { fmt.Println(terminal.Detect(os.Getenv).Name()) }

func runDetect(cfg config.Config) {
	term := terminal.Detect(os.Getenv)
	c := term.Capabilities()
	yn := func(b bool) string {
		if b {
			return "yes"
		}
		return "no"
	}
	fmt.Printf("terminal:   %s\ntab color:  %s\nbackground: %s\ntitle:      %s\ntruecolor:  %s\ntmux:       %s\n",
		term.Name(), yn(c.TabColor), yn(c.Background), yn(c.Title), yn(c.TrueColor), yn(terminal.InTmux(os.Getenv)))
	for _, s := range []string{config.Idle, config.Working, config.Attention, config.Done, config.Error} {
		st := cfg.States[s]
		t, _ := colors.ParseHex(st.Tab)
		b, _ := colors.ParseHex(st.Background)
		fmt.Printf("%-10s  %s  tab %s %s  bg %s\n", s, st.Glyph, bg(t)+"   "+sgrReset, st.Tab, bg(b)+"   "+sgrReset)
	}
}

const sgrReset = "\x1b[0m"

func bg(c colors.Color) string { return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", c.R, c.G, c.B) }
func fg(c colors.Color) string { return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", c.R, c.G, c.B) }
