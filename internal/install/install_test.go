package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CassianFlorin/skill-hub/internal/audit"
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

func TestInstallRejectsUnsatisfiedSkillhubRequirement(t *testing.T) {
	t.Setenv("SKILLHUB_HOME", t.TempDir())
	workDir := t.TempDir()
	skillDir := filepath.Join(t.TempDir(), "review")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	yaml := "name: review\nnamespace: acme\nversion: 1.0.0\ndescription: d\ntargets:\n- codex\nrequires:\n  skillhub: \">=9.0.0\"\n"
	if err := os.WriteFile(filepath.Join(skillDir, "skill.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatalf("write skill.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# review\n"), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	previousVersion := SkillhubVersion
	t.Cleanup(func() { SkillhubVersion = previousVersion })

	SkillhubVersion = "v1.3.11"
	if _, err := Install(workDir, skillDir); err == nil || !strings.Contains(err.Error(), "requires skillhub >=9.0.0") {
		t.Errorf("expected requirement error, got %v", err)
	}

	SkillhubVersion = "dev"
	if _, err := Install(workDir, skillDir); err != nil {
		t.Errorf("dev build should skip requirement check, got %v", err)
	}

	SkillhubVersion = "v9.1.0"
	t.Setenv("SKILLHUB_HOME", t.TempDir())
	if _, err := Install(workDir, skillDir); err != nil {
		t.Errorf("satisfied requirement should install, got %v", err)
	}
}

func writeUpdatePolicySkill(t *testing.T, dir string, version string, breaking bool) {
	t.Helper()
	yaml := "name: policy\nnamespace: acme\nversion: " + version + "\ndescription: d\ntargets:\n- codex\n"
	if breaking {
		yaml += "compatibility:\n  breaking: true\n"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "skill.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatalf("write skill.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# policy\n"), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
}

func TestUpdateSkipsMajorWithoutConfirmation(t *testing.T) {
	t.Setenv("SKILLHUB_HOME", t.TempDir())
	workDir := t.TempDir()
	sourceDir := filepath.Join(t.TempDir(), "policy")
	writeUpdatePolicySkill(t, sourceDir, "1.0.0", false)
	if _, err := Install(workDir, sourceDir); err != nil {
		t.Fatalf("install: %v", err)
	}

	writeUpdatePolicySkill(t, sourceDir, "2.0.0", false)
	changes, skipped, err := Update(UpdateOptions{})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(changes) != 0 {
		t.Errorf("major update applied without --major: %v", changes)
	}
	if len(skipped) != 1 || skipped[0].Reason != "major update" || skipped[0].AvailableVersion != "2.0.0" {
		t.Fatalf("skipped = %+v", skipped)
	}

	changes, skipped, err = Update(UpdateOptions{AllowMajor: true})
	if err != nil {
		t.Fatalf("update --major: %v", err)
	}
	if len(changes) != 1 || len(skipped) != 0 {
		t.Fatalf("changes = %v, skipped = %v", changes, skipped)
	}
	lock, err := LoadLock()
	if err != nil {
		t.Fatalf("LoadLock: %v", err)
	}
	if lock.Skills[0].Version != "2.0.0" {
		t.Errorf("lock version = %q, want 2.0.0", lock.Skills[0].Version)
	}
}

func TestUpdateSkipsBreakingMinorWithoutConfirmation(t *testing.T) {
	t.Setenv("SKILLHUB_HOME", t.TempDir())
	workDir := t.TempDir()
	sourceDir := filepath.Join(t.TempDir(), "policy")
	writeUpdatePolicySkill(t, sourceDir, "1.0.0", false)
	if _, err := Install(workDir, sourceDir); err != nil {
		t.Fatalf("install: %v", err)
	}

	writeUpdatePolicySkill(t, sourceDir, "1.1.0", true)
	changes, skipped, err := Update(UpdateOptions{})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(changes) != 0 || len(skipped) != 1 || skipped[0].Reason != "breaking change" {
		t.Fatalf("changes = %v, skipped = %+v", changes, skipped)
	}

	changes, _, err = Update(UpdateOptions{AllowMajor: true})
	if err != nil {
		t.Fatalf("update --major: %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("breaking update not applied with --major: %v", changes)
	}
}

func TestUpdateAppliesMinorAutomatically(t *testing.T) {
	t.Setenv("SKILLHUB_HOME", t.TempDir())
	workDir := t.TempDir()
	sourceDir := filepath.Join(t.TempDir(), "policy")
	writeUpdatePolicySkill(t, sourceDir, "1.0.0", false)
	if _, err := Install(workDir, sourceDir); err != nil {
		t.Fatalf("install: %v", err)
	}

	writeUpdatePolicySkill(t, sourceDir, "1.2.3", false)
	changes, skipped, err := Update(UpdateOptions{})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(changes) != 1 || len(skipped) != 0 {
		t.Fatalf("changes = %v, skipped = %v", changes, skipped)
	}
}

func TestLifecycleWritesAuditEvents(t *testing.T) {
	t.Setenv("SKILLHUB_HOME", t.TempDir())
	workDir := t.TempDir()
	sourceDir := filepath.Join(t.TempDir(), "policy")
	writeUpdatePolicySkill(t, sourceDir, "1.0.0", false)
	if _, err := Install(workDir, sourceDir); err != nil {
		t.Fatalf("install: %v", err)
	}
	writeUpdatePolicySkill(t, sourceDir, "1.1.0", false)
	if _, _, err := Update(UpdateOptions{}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if _, err := Rollback("acme/policy"); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if _, err := Uninstall("acme/policy"); err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if _, err := Install(workDir, filepath.Join(sourceDir, "missing")); err == nil {
		t.Fatal("expected failing install")
	}

	events, err := audit.List(0)
	if err != nil {
		t.Fatalf("audit.List: %v", err)
	}
	var commands []string
	for _, event := range events {
		commands = append(commands, event.Command+":"+event.Result)
	}
	want := []string{"install:ok", "update:ok", "rollback:ok", "uninstall:ok", "install:error"}
	if len(commands) != len(want) {
		t.Fatalf("audit commands = %v, want %v", commands, want)
	}
	for i := range want {
		if commands[i] != want[i] {
			t.Errorf("audit[%d] = %q, want %q", i, commands[i], want[i])
		}
	}
	if events[1].FromVersion != "1.0.0" || events[1].Version != "1.1.0" {
		t.Errorf("update audit event = %+v", events[1])
	}
	if events[2].FromVersion != "1.1.0" || events[2].Version != "1.0.0" {
		t.Errorf("rollback audit event = %+v", events[2])
	}
}
