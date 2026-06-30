package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

// installSampleSkill scaffolds a local registry holding one skill, then inits a
// project, registers the registry, and installs the skill. It returns the home
// and project directories so callers can drive --json commands against them.
func installSampleSkill(t *testing.T, version string) (home string, projectDir string, skillDir string) {
	t.Helper()
	workspace := t.TempDir()
	home = filepath.Join(workspace, "skillhub-home")
	projectDir = filepath.Join(workspace, "project")
	registryDir := filepath.Join(workspace, "registry")
	skillDir = filepath.Join(registryDir, "java-review")
	t.Setenv("SKILLHUB_HOME", home)
	// Isolate runtime directories so deploy never touches the real ~/.codex etc.
	t.Setenv("SKILLHUB_CODEX_DIR", filepath.Join(workspace, "rt-codex"))
	t.Setenv("SKILLHUB_CLAUDE_DIR", filepath.Join(workspace, "rt-claude"))
	t.Setenv("SKILLHUB_GEMINI_DIR", filepath.Join(workspace, "rt-gemini"))
	t.Setenv("SKILLHUB_HERMES_DIR", filepath.Join(workspace, "rt-hermes"))
	t.Setenv("SKILLHUB_HERMES_HOME", filepath.Join(workspace, "rt-hermes-home"))

	mustWriteFile(t, filepath.Join(skillDir, "skill.yaml"), strings.TrimSpace(`
name: java-review
namespace: platform
version: `+version+`
description: Java review skill
entry: SKILL.md
targets:
  - codex
`)+"\n")
	mustWriteFile(t, filepath.Join(skillDir, "SKILL.md"), "# Java Review\n\nv"+version+"\n")

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "init")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "registry", "add", "local", "company", registryDir)
	stdout.Reset()
	runOK(t, projectDir, &stdout, "install", "company/java-review")
	return home, projectDir, skillDir
}

func TestListJSONReportsInstalledSkills(t *testing.T) {
	_, projectDir, _ := installSampleSkill(t, "1.2.0")

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "list", "--json")

	var rows []map[string]any
	mustUnmarshalJSON(t, stdout.String(), &rows)
	if len(rows) != 1 {
		t.Fatalf("expected one list row, got %#v", rows)
	}
	if rows[0]["scope"] != "global" || rows[0]["skill"] != "platform/java-review" || rows[0]["version"] != "1.2.0" {
		t.Fatalf("unexpected list row: %#v", rows[0])
	}
}

func TestDoctorJSONReportsConfigRuntimesAndRegistries(t *testing.T) {
	_, projectDir, _ := installSampleSkill(t, "1.2.0")

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "doctor", "--json")

	var doc map[string]any
	mustUnmarshalJSON(t, stdout.String(), &doc)
	if doc["config"] != "ok" {
		t.Fatalf("expected config ok, got %#v", doc["config"])
	}
	if installed, ok := doc["installed"].(float64); !ok || int(installed) != 1 {
		t.Fatalf("expected installed=1, got %#v", doc["installed"])
	}
	runtimes, ok := doc["runtimes"].([]any)
	if !ok || len(runtimes) == 0 {
		t.Fatalf("expected runtimes array, got %#v", doc["runtimes"])
	}
	registries, ok := doc["registries"].([]any)
	if !ok || len(registries) == 0 {
		t.Fatalf("expected registries array, got %#v", doc["registries"])
	}
}

func TestRegistryListJSONReportsConfiguredRegistries(t *testing.T) {
	_, projectDir, _ := installSampleSkill(t, "1.2.0")

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "registry", "list", "--json")

	var rows []map[string]any
	mustUnmarshalJSON(t, stdout.String(), &rows)
	found := false
	for _, row := range rows {
		if row["name"] == "company" && row["type"] == "local" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected company registry in JSON, got %#v", rows)
	}
}

func TestCheckJSONReportsAvailableUpdates(t *testing.T) {
	_, projectDir, skillDir := installSampleSkill(t, "1.2.0")
	mustWriteFile(t, filepath.Join(skillDir, "skill.yaml"), strings.Replace(readFile(t, filepath.Join(skillDir, "skill.yaml")), "version: 1.2.0", "version: 1.3.0", 1))

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "check", "--json")

	var rows []map[string]any
	mustUnmarshalJSON(t, stdout.String(), &rows)
	if len(rows) != 1 {
		t.Fatalf("expected one update plan, got %#v", rows)
	}
	if rows[0]["identity"] != "platform/java-review" || rows[0]["current_version"] != "1.2.0" || rows[0]["available_version"] != "1.3.0" {
		t.Fatalf("unexpected update plan: %#v", rows[0])
	}
}

func TestDeployStatusJSONReportsRuntimeState(t *testing.T) {
	_, projectDir, _ := installSampleSkill(t, "1.2.0")

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "deploy", "codex")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "deploy", "status", "--json")

	var rows []map[string]any
	mustUnmarshalJSON(t, stdout.String(), &rows)
	found := false
	for _, row := range rows {
		if row["identity"] == "platform/java-review" && row["runtime"] == "codex" && row["state"] == "deployed" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected deployed codex status, got %#v", rows)
	}
}

func TestHoldsAndHistoryJSON(t *testing.T) {
	_, projectDir, skillDir := installSampleSkill(t, "1.2.0")

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "hold", "platform/java-review", "--reason", "pinned")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "holds", "--json")
	var holds []map[string]any
	mustUnmarshalJSON(t, stdout.String(), &holds)
	if len(holds) != 1 || holds[0]["skill"] != "platform/java-review" || holds[0]["reason"] != "pinned" {
		t.Fatalf("unexpected holds JSON: %#v", holds)
	}

	// Produce a rollback history entry by updating the held skill after unhold.
	stdout.Reset()
	runOK(t, projectDir, &stdout, "unhold", "platform/java-review")
	mustWriteFile(t, filepath.Join(skillDir, "skill.yaml"), strings.Replace(readFile(t, filepath.Join(skillDir, "skill.yaml")), "version: 1.2.0", "version: 1.3.0", 1))
	stdout.Reset()
	runOK(t, projectDir, &stdout, "update", "platform/java-review")

	stdout.Reset()
	runOK(t, projectDir, &stdout, "history", "platform/java-review", "--json")
	var history []map[string]any
	mustUnmarshalJSON(t, stdout.String(), &history)
	if len(history) == 0 {
		t.Fatalf("expected history entries, got %#v", history)
	}
	if history[0]["identity"] != "platform/java-review" {
		t.Fatalf("unexpected history JSON: %#v", history[0])
	}
}
