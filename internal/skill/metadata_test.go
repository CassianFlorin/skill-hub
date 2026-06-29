package skill

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIdentity(t *testing.T) {
	if got := Identity("official", "git-commit"); got != "official/git-commit" {
		t.Fatalf("Identity = %q, want official/git-commit", got)
	}
}

func TestSafeIdentity(t *testing.T) {
	cases := map[string]string{
		"official/git-commit": "official__git-commit",
		"team\\sub/name":      "team__sub__name",
		"ns:name with space":  "ns_name-with-space",
	}
	for in, want := range cases {
		if got := SafeIdentity(in); got != want {
			t.Errorf("SafeIdentity(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestResolveNamespace(t *testing.T) {
	if got := ResolveNamespace(Metadata{Namespace: "ns"}, "fallback"); got != "ns" {
		t.Errorf("namespace precedence: got %q, want ns", got)
	}
	if got := ResolveNamespace(Metadata{Author: "alice"}, "fallback"); got != "alice" {
		t.Errorf("author fallback: got %q, want alice", got)
	}
	if got := ResolveNamespace(Metadata{}, "fallback"); got != "fallback" {
		t.Errorf("explicit fallback: got %q, want fallback", got)
	}
}

func writeSkill(t *testing.T, dir string, yaml string, entry string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "skill.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	if entry != "" {
		if err := os.WriteFile(filepath.Join(dir, entry), []byte("# entry\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLoadMetadata(t *testing.T) {
	dir := t.TempDir()
	writeSkill(t, dir, "name: git-commit\nnamespace: official\nversion: 1.2.0\ntargets:\n  - codex\n  - claude\ntags:\n  - git\n", "SKILL.md")

	meta, err := LoadMetadata(dir)
	if err != nil {
		t.Fatalf("LoadMetadata: %v", err)
	}
	if meta.Name != "git-commit" || meta.Namespace != "official" || meta.Version != "1.2.0" {
		t.Errorf("unexpected scalar fields: %+v", meta)
	}
	if len(meta.Targets) != 2 || meta.Targets[0] != "codex" || meta.Targets[1] != "claude" {
		t.Errorf("targets = %v, want [codex claude]", meta.Targets)
	}
	if len(meta.Tags) != 1 || meta.Tags[0] != "git" {
		t.Errorf("tags = %v, want [git]", meta.Tags)
	}
}

func TestLoadMetadataDefaultsVersion(t *testing.T) {
	dir := t.TempDir()
	writeSkill(t, dir, "name: thing\n", "SKILL.md")
	meta, err := LoadMetadata(dir)
	if err != nil {
		t.Fatalf("LoadMetadata: %v", err)
	}
	if meta.Version != Unversioned {
		t.Errorf("version = %q, want %q", meta.Version, Unversioned)
	}
}

func TestLoadMetadataMissingName(t *testing.T) {
	dir := t.TempDir()
	writeSkill(t, dir, "namespace: official\n", "SKILL.md")
	if _, err := LoadMetadata(dir); err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestLoadMetadataMissingEntry(t *testing.T) {
	dir := t.TempDir()
	// declare an entry file that does not exist on disk
	writeSkill(t, dir, "name: thing\nentry: MISSING.md\n", "")
	if _, err := LoadMetadata(dir); err == nil {
		t.Fatal("expected error for missing entry file")
	}
}

func TestLoadCompatibleMetadataLegacy(t *testing.T) {
	dir := t.TempDir()
	// no skill.yaml, only SKILL.md -> generated metadata
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# legacy\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	meta, err := LoadCompatibleMetadata(dir, "fallback-ns")
	if err != nil {
		t.Fatalf("LoadCompatibleMetadata: %v", err)
	}
	if !meta.Generated {
		t.Error("expected Generated = true for legacy skill")
	}
	if meta.Name != filepath.Base(dir) {
		t.Errorf("name = %q, want dir base %q", meta.Name, filepath.Base(dir))
	}
	if meta.Namespace != "fallback-ns" {
		t.Errorf("namespace = %q, want fallback-ns", meta.Namespace)
	}
}

func TestWriteGeneratedMetadataRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# entry\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	in := Metadata{Name: "thing", Namespace: "ns", Version: "9.9.9", Entry: "SKILL.md"}
	if err := WriteGeneratedMetadata(dir, in); err != nil {
		t.Fatalf("WriteGeneratedMetadata: %v", err)
	}
	out, err := LoadMetadata(dir)
	if err != nil {
		t.Fatalf("LoadMetadata after write: %v", err)
	}
	if out.Name != in.Name || out.Namespace != in.Namespace || out.Version != in.Version {
		t.Errorf("round trip mismatch: wrote %+v, read %+v", in, out)
	}
	if !out.Generated {
		t.Error("expected Generated = true after WriteGeneratedMetadata")
	}
}
