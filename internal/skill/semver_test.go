package skill

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompareSemver(t *testing.T) {
	cases := []struct {
		left       string
		right      string
		cmp        int
		comparable bool
	}{
		{"1.0.0", "1.0.0", 0, true},
		{"v1.0.0", "1.0.0", 0, true},
		{"1.0.1", "1.0.0", 1, true},
		{"1.2.0", "1.10.0", -1, true},
		{"2.0.0-rc.1", "2.0.0", -1, true},
		{"2.0.0", "2.0.0-rc.1", 1, true},
		{"2.0.0-alpha", "2.0.0-beta", -1, true},
		{"1.0.0+build.5", "1.0.0", 0, true},
		{"latest", "1.0.0", 0, false},
		{"1.0", "1.0.0", 0, false},
		{"1.0.x", "1.0.0", 0, false},
	}
	for _, testCase := range cases {
		cmp, comparable := CompareSemver(testCase.left, testCase.right)
		if cmp != testCase.cmp || comparable != testCase.comparable {
			t.Errorf("CompareSemver(%q, %q) = (%d, %v), want (%d, %v)", testCase.left, testCase.right, cmp, comparable, testCase.cmp, testCase.comparable)
		}
	}
}

func TestSatisfiesConstraint(t *testing.T) {
	cases := []struct {
		version    string
		constraint string
		want       bool
	}{
		{"1.3.11", ">=1.3.0", true},
		{"1.2.9", ">=1.3.0", false},
		{"1.3.0", ">1.3.0", false},
		{"1.3.1", ">1.3.0", true},
		{"1.3.0", "<=1.3.0", true},
		{"1.4.0", "<2.0.0", true},
		{"2.0.0", "<2.0.0", false},
		{"1.3.0", "1.3.0", true},
		{"1.3.1", "=1.3.0", false},
		{"1.5.0", ">=1.3.0, <2.0.0", true},
		{"2.1.0", ">=1.3.0, <2.0.0", false},
		{"v1.3.11", ">=1.3.0", true},
		{"dev", ">=1.3.0", false},
	}
	for _, testCase := range cases {
		got, err := SatisfiesConstraint(testCase.version, testCase.constraint)
		if err != nil {
			t.Errorf("SatisfiesConstraint(%q, %q): %v", testCase.version, testCase.constraint, err)
			continue
		}
		if got != testCase.want {
			t.Errorf("SatisfiesConstraint(%q, %q) = %v, want %v", testCase.version, testCase.constraint, got, testCase.want)
		}
	}
}

func TestValidateConstraint(t *testing.T) {
	for _, valid := range []string{">=1.0.0", "1.2.3", ">=1.0.0, <2.0.0", "=1.0.0", "<=2.0.0-rc.1"} {
		if err := ValidateConstraint(valid); err != nil {
			t.Errorf("ValidateConstraint(%q): %v", valid, err)
		}
	}
	for _, invalid := range []string{"", ">=latest", "~1.0.0", ">=1.0", "1.0.0 || 2.0.0"} {
		if err := ValidateConstraint(invalid); err == nil {
			t.Errorf("ValidateConstraint(%q) expected error", invalid)
		}
	}
}

func TestLoadMetadataRequiresBlock(t *testing.T) {
	dir := t.TempDir()
	yaml := `name: demo
namespace: acme
version: 1.0.0
description: Demo skill
targets:
- codex
requires:
  skillhub: ">=1.4.0"
  codex: ">=0.5.0, <1.0.0"
tags:
- demo
`
	if err := os.WriteFile(filepath.Join(dir, "skill.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatalf("write skill.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# demo\n"), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	meta, err := LoadMetadata(dir)
	if err != nil {
		t.Fatalf("LoadMetadata: %v", err)
	}
	if meta.Requires["skillhub"] != ">=1.4.0" {
		t.Errorf("requires.skillhub = %q", meta.Requires["skillhub"])
	}
	if meta.Requires["codex"] != ">=0.5.0, <1.0.0" {
		t.Errorf("requires.codex = %q", meta.Requires["codex"])
	}
	if len(meta.Tags) != 1 || meta.Tags[0] != "demo" {
		t.Errorf("tags after requires block = %v", meta.Tags)
	}
	if len(meta.Targets) != 1 || meta.Targets[0] != "codex" {
		t.Errorf("targets = %v", meta.Targets)
	}
}

func TestLoadMetadataRequiresEntryWithoutValueFails(t *testing.T) {
	dir := t.TempDir()
	yaml := "name: demo\nversion: 1.0.0\nrequires:\n  skillhub:\n"
	if err := os.WriteFile(filepath.Join(dir, "skill.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatalf("write skill.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# demo\n"), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	if _, err := LoadMetadata(dir); err == nil {
		t.Error("expected error for requires entry without constraint")
	}
}
