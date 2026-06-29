package project

import (
	"os"
	"path/filepath"
	"testing"
)

func makeSkill(t *testing.T, dir string, name string, namespace string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# "+name+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	yaml := "name: " + name + "\nnamespace: " + namespace + "\nversion: 1.0.0\n"
	if err := os.WriteFile(filepath.Join(dir, "skill.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverSkillsEmpty(t *testing.T) {
	found, err := DiscoverSkills(t.TempDir())
	if err != nil {
		t.Fatalf("DiscoverSkills: %v", err)
	}
	if len(found) != 0 {
		t.Errorf("expected no skills, got %v", found)
	}
}

func TestDiscoverSkillsAcrossRoots(t *testing.T) {
	work := t.TempDir()
	makeSkill(t, filepath.Join(work, ".codex", "skills", "alpha"), "alpha", "team")
	makeSkill(t, filepath.Join(work, ".claude", "skills", "beta"), "beta", "team")
	// a directory without SKILL.md must be ignored
	if err := os.MkdirAll(filepath.Join(work, ".claude", "skills", "not-a-skill"), 0o755); err != nil {
		t.Fatal(err)
	}

	found, err := DiscoverSkills(work)
	if err != nil {
		t.Fatalf("DiscoverSkills: %v", err)
	}
	if len(found) != 2 {
		t.Fatalf("expected 2 skills, got %d: %+v", len(found), found)
	}
	// results are sorted by identity, so alpha precedes beta
	if found[0].Identity != "team/alpha" || found[1].Identity != "team/beta" {
		t.Errorf("unexpected identities/order: %+v", found)
	}
	if found[0].RelPath != ".codex/skills/alpha" {
		t.Errorf("RelPath = %q, want .codex/skills/alpha", found[0].RelPath)
	}
}

func TestDiscoverSkillsLegacyWithoutYAML(t *testing.T) {
	work := t.TempDir()
	dir := filepath.Join(work, ".skillhub", "skills", "legacy")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// only SKILL.md, no skill.yaml -> compatible metadata with "project" fallback namespace
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# legacy\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	found, err := DiscoverSkills(work)
	if err != nil {
		t.Fatalf("DiscoverSkills: %v", err)
	}
	if len(found) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(found))
	}
	if found[0].Identity != "project/legacy" {
		t.Errorf("Identity = %q, want project/legacy", found[0].Identity)
	}
}
