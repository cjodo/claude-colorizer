package setup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readJSON(t *testing.T, p string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	m := map[string]any{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestInstallUninstallRoundTrip(t *testing.T) {
	dir := t.TempDir()
	settings := filepath.Join(dir, "settings.json")
	orig := `{
  "model": "opus",
  "statusLine": {"type": "command", "command": "~/bin/my status.sh"},
  "hooks": {"Stop": [{"hooks": [{"type": "command", "command": "notify-send done"}]}]}
}`
	if err := os.WriteFile(settings, []byte(orig), 0o600); err != nil {
		t.Fatal(err)
	}
	o := Options{Settings: settings, Binary: "/opt/my bin/claude-colorizer", StateDir: filepath.Join(dir, "state"), Hooks: true, Statusline: true}

	if _, err := Install(o); err != nil {
		t.Fatal(err)
	}
	// Second install must not duplicate anything.
	if _, err := Install(o); err != nil {
		t.Fatal(err)
	}

	s := readJSON(t, settings)
	cmd := s["statusLine"].(map[string]any)["command"].(string)
	if want := `'/opt/my bin/claude-colorizer' statusline -- sh -c '~/bin/my status.sh'`; cmd != want {
		t.Errorf("statusLine = %q, want %q", cmd, want)
	}
	stop := s["hooks"].(map[string]any)["Stop"].([]any)
	if len(stop) != 2 {
		t.Errorf("Stop groups = %d, want 2 (user's + ours)", len(stop))
	}
	if len(s["hooks"].(map[string]any)) != len(hookEvents) {
		t.Errorf("hook events = %d, want %d", len(s["hooks"].(map[string]any)), len(hookEvents))
	}
	if s["model"] != "opus" {
		t.Error("unrelated settings were lost")
	}
	if _, err := os.Stat(settings + ".bak"); err != nil {
		t.Error("no backup written")
	}

	if _, err := Uninstall(o); err != nil {
		t.Fatal(err)
	}
	s = readJSON(t, settings)
	if got := s["statusLine"].(map[string]any)["command"]; got != "~/bin/my status.sh" {
		t.Errorf("statusLine not restored: %v", got)
	}
	hooks := s["hooks"].(map[string]any)
	if len(hooks) != 1 || len(hooks["Stop"].([]any)) != 1 {
		t.Errorf("user hooks not preserved exactly: %v", hooks)
	}
	b, _ := json.Marshal(s)
	if strings.Contains(string(b), marker) {
		t.Errorf("leftover entries: %s", b)
	}
}

func TestInstallFreshAndInvalid(t *testing.T) {
	dir := t.TempDir()
	settings := filepath.Join(dir, "nested", "settings.json")
	o := Options{Settings: settings, Binary: "/usr/bin/claude-colorizer", StateDir: dir, Hooks: true, Statusline: true}
	if _, err := Install(o); err != nil {
		t.Fatal(err)
	}
	if cmd := readJSON(t, settings)["statusLine"].(map[string]any)["command"]; cmd != "/usr/bin/claude-colorizer statusline" {
		t.Errorf("statusLine = %v", cmd)
	}

	bad := filepath.Join(dir, "bad.json")
	os.WriteFile(bad, []byte("{ // comment\n}"), 0o600)
	o.Settings = bad
	if _, err := Install(o); err == nil {
		t.Error("expected error for invalid JSON")
	}
	if b, _ := os.ReadFile(bad); string(b) != "{ // comment\n}" {
		t.Error("invalid file was modified")
	}
}
