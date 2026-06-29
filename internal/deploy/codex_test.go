package deploy

import (
	"path/filepath"
	"testing"

	"github.com/cassian/skill-hub/internal/install"
)

func TestRuntimeNames(t *testing.T) {
	names := RuntimeNames()
	want := map[string]bool{"codex": false, "claude": false, "gemini": false, "hermes": false}
	for _, n := range names {
		if _, ok := want[n]; !ok {
			t.Errorf("unexpected runtime %q", n)
		}
		want[n] = true
	}
	for n, seen := range want {
		if !seen {
			t.Errorf("missing runtime %q", n)
		}
	}
}

func TestRuntimeDirFromEnv(t *testing.T) {
	t.Setenv("SKILLHUB_CODEX_DIR", "/tmp/codex-skills")
	dir, err := RuntimeDir("codex")
	if err != nil {
		t.Fatalf("RuntimeDir: %v", err)
	}
	want, _ := filepath.Abs("/tmp/codex-skills")
	if dir != want {
		t.Errorf("RuntimeDir(codex) = %q, want %q", dir, want)
	}
}

func TestRuntimeDirHermesProfile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SKILLHUB_HERMES_HOME", home)
	dir, err := RuntimeDirWithOptions("hermes", Options{Profile: "work"})
	if err != nil {
		t.Fatalf("RuntimeDirWithOptions: %v", err)
	}
	want := filepath.Join(home, "profiles", "work", "skills")
	if dir != want {
		t.Errorf("hermes profile dir = %q, want %q", dir, want)
	}
}

func TestRuntimeDirUnsupported(t *testing.T) {
	if _, err := RuntimeDir("bogus"); err == nil {
		t.Fatal("expected error for unsupported runtime")
	}
}

func TestRuntimeTarget(t *testing.T) {
	t.Setenv("SKILLHUB_CLAUDE_DIR", "/tmp/claude")
	got, err := RuntimeTarget("claude", "official/git-commit")
	if err != nil {
		t.Fatalf("RuntimeTarget: %v", err)
	}
	root, _ := filepath.Abs("/tmp/claude")
	want := filepath.Join(root, "official__git-commit")
	if got != want {
		t.Errorf("RuntimeTarget = %q, want %q", got, want)
	}
}

func TestSupportsRuntime(t *testing.T) {
	// no targets declared => supports everything (legacy packages)
	if !supportsRuntime(install.LockedSkill{}, "codex") {
		t.Error("empty targets should support any runtime")
	}
	locked := install.LockedSkill{Targets: []string{"codex", "claude"}}
	if !supportsRuntime(locked, "claude") {
		t.Error("declared target claude should be supported")
	}
	if supportsRuntime(locked, "gemini") {
		t.Error("undeclared target gemini should not be supported")
	}
}

func TestAppendRuntimeDedupes(t *testing.T) {
	got := appendRuntime([]string{"codex"}, "codex")
	if len(got) != 1 {
		t.Errorf("appendRuntime duplicated entry: %v", got)
	}
	got = appendRuntime([]string{"codex"}, "claude")
	if len(got) != 2 || got[1] != "claude" {
		t.Errorf("appendRuntime did not add new runtime: %v", got)
	}
}
