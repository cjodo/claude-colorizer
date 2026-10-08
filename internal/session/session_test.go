package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvDir, dir)

	if got := Load("abc-123"); got != "" {
		t.Errorf("unknown session = %q, want empty", got)
	}
	if err := Save("abc-123", "working"); err != nil {
		t.Fatal(err)
	}
	if err := Save("abc-123", "done"); err != nil {
		t.Fatal(err)
	}
	if got := Load("abc-123"); got != "done" {
		t.Errorf("Load = %q, want done", got)
	}
	if err := Save("abc-123", ""); err != nil {
		t.Fatal(err)
	}
	if got := Load("abc-123"); got != "" {
		t.Errorf("after forget = %q, want empty", got)
	}
	if err := Save("abc-123", ""); err != nil {
		t.Errorf("forgetting twice: %v", err)
	}
	if left, _ := os.ReadDir(dir); len(left) != 0 {
		t.Errorf("leftover files: %v", left)
	}
}

func TestRejectsUnsafeIDs(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvDir, filepath.Join(dir, "sessions"))
	for _, id := range []string{"", "../escape", "a/b", "."} {
		if err := Save(id, "working"); err != nil {
			t.Errorf("Save(%q): %v", id, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "escape")); err == nil {
		t.Error("path traversal wrote outside the state dir")
	}
	if _, err := os.Stat(filepath.Join(dir, "sessions")); err == nil {
		t.Error("unsafe ids should not create anything")
	}
}
