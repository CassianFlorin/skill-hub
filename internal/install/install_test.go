package install

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDisplayIdentity(t *testing.T) {
	cases := []struct {
		skill LockedSkill
		want  string
	}{
		{LockedSkill{Identity: "official/git"}, "official/git"},
		{LockedSkill{Namespace: "team", Name: "review"}, "team/review"},
		{LockedSkill{Name: "loner"}, "loner"},
	}
	for _, c := range cases {
		if got := c.skill.DisplayIdentity(); got != c.want {
			t.Errorf("DisplayIdentity(%+v) = %q, want %q", c.skill, got, c.want)
		}
	}
}

func TestLockFileFind(t *testing.T) {
	lock := LockFile{Skills: []LockedSkill{
		{Identity: "official/git", Name: "git"},
		{Namespace: "team", Name: "review"},
	}}
	if _, ok := lock.find("official/git"); !ok {
		t.Error("find by identity failed")
	}
	if _, ok := lock.find("git"); !ok {
		t.Error("find by name failed")
	}
	if _, ok := lock.find("team/review"); !ok {
		t.Error("find by composed identity failed")
	}
	if _, ok := lock.find("missing"); ok {
		t.Error("find returned a match for a missing identity")
	}
}

func TestLockFileUpsertAndRemove(t *testing.T) {
	lock := LockFile{}
	lock.upsert(LockedSkill{Identity: "a/x", Version: "1.0.0"})
	lock.upsert(LockedSkill{Identity: "b/y", Version: "1.0.0"})
	if len(lock.Skills) != 2 {
		t.Fatalf("expected 2 skills after inserts, got %d", len(lock.Skills))
	}
	// upsert with same identity replaces in place
	lock.upsert(LockedSkill{Identity: "a/x", Version: "2.0.0"})
	if len(lock.Skills) != 2 {
		t.Fatalf("upsert duplicated identity: %d skills", len(lock.Skills))
	}
	found, ok := lock.find("a/x")
	if !ok || found.Version != "2.0.0" {
		t.Errorf("upsert did not update version: %+v", found)
	}
	lock.remove("a/x")
	if _, ok := lock.find("a/x"); ok {
		t.Error("remove did not delete identity")
	}
	if len(lock.Skills) != 1 {
		t.Errorf("expected 1 skill after remove, got %d", len(lock.Skills))
	}
}

func TestSaveLoadLockRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SKILLHUB_HOME", home)

	// empty load when no lockfile exists yet
	empty, err := LoadLock()
	if err != nil {
		t.Fatalf("LoadLock (missing): %v", err)
	}
	if len(empty.Skills) != 0 {
		t.Errorf("expected empty lock, got %+v", empty)
	}

	in := LockFile{Skills: []LockedSkill{
		{Identity: "official/git", Name: "git", Version: "1.0.0", Targets: []string{"codex"}},
	}}
	if err := SaveLock(in); err != nil {
		t.Fatalf("SaveLock: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, LockFileName)); err != nil {
		t.Fatalf("lockfile not written: %v", err)
	}

	out, err := LoadLock()
	if err != nil {
		t.Fatalf("LoadLock: %v", err)
	}
	if len(out.Skills) != 1 || out.Skills[0].Identity != "official/git" || out.Skills[0].Version != "1.0.0" {
		t.Errorf("round trip mismatch: %+v", out.Skills)
	}
}

func TestInferCachedSource(t *testing.T) {
	// empty path is never inferable
	if _, _, ok := inferCachedSource("", ""); ok {
		t.Error("empty source path should not be inferable")
	}

	// build a fake git cache: <root>/myreg/.git and a skill subdir
	root := t.TempDir()
	repo := filepath.Join(root, "myreg")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	skillPath := filepath.Join(repo, "skills", "git-commit")
	if err := os.MkdirAll(skillPath, 0o755); err != nil {
		t.Fatal(err)
	}
	name, subpath, ok := inferCachedSource(skillPath, "")
	if !ok {
		t.Fatal("expected cached source to be inferable")
	}
	if name != "myreg" {
		t.Errorf("cache name = %q, want myreg", name)
	}
	if subpath != "skills/git-commit" {
		t.Errorf("subpath = %q, want skills/git-commit", subpath)
	}
}
