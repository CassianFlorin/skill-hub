package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMVPWorkflowInstallsListsUpdatesAndDeploysLocalSkills(t *testing.T) {
	workspace := t.TempDir()
	home := filepath.Join(workspace, "skillhub-home")
	codexDir := filepath.Join(workspace, "codex-skills")
	registryDir := filepath.Join(workspace, "registry")
	skillDir := filepath.Join(registryDir, "java-review")
	projectDir := filepath.Join(workspace, "project")

	mustWriteFile(t, filepath.Join(skillDir, "skill.yaml"), strings.TrimSpace(`
name: java-review
version: 1.2.0
description: Java review skill
author: platform-team
entry: SKILL.md
targets:
  - codex
  - claude
tags:
  - java
  - review
`)+"\n")
	mustWriteFile(t, filepath.Join(skillDir, "SKILL.md"), "# Java Review\n\nReview Java changes.\n")
	mustWriteFile(t, filepath.Join(skillDir, "references", "rules.md"), "Prefer focused reviews.\n")
	t.Setenv("SKILLHUB_HOME", home)
	t.Setenv("SKILLHUB_CODEX_DIR", codexDir)

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "init")
	assertFileContains(t, filepath.Join(projectDir, "skillhub.yaml"), `"install_dir"`)

	stdout.Reset()
	runOK(t, projectDir, &stdout, "registry", "add", "local", "company", registryDir)
	assertContains(t, stdout.String(), "registered company")
	assertFileContains(t, filepath.Join(projectDir, "skillhub.yaml"), `"company"`)

	stdout.Reset()
	runOK(t, projectDir, &stdout, "install", "company/java-review")
	assertContains(t, stdout.String(), "installed java-review@1.2.0")
	assertFileContains(t, filepath.Join(home, "installed", "java-review", "SKILL.md"), "Review Java changes")
	assertFileContains(t, filepath.Join(home, "skillhub.lock"), `"version": "1.2.0"`)

	stdout.Reset()
	runOK(t, projectDir, &stdout, "list")
	assertContains(t, stdout.String(), "java-review")
	assertContains(t, stdout.String(), "1.2.0")

	mustWriteFile(t, filepath.Join(skillDir, "skill.yaml"), strings.Replace(readFile(t, filepath.Join(skillDir, "skill.yaml")), "version: 1.2.0", "version: 1.3.0", 1))
	stdout.Reset()
	runOK(t, projectDir, &stdout, "update")
	assertContains(t, stdout.String(), "updated java-review 1.2.0 -> 1.3.0")
	assertFileContains(t, filepath.Join(home, "skillhub.lock"), `"version": "1.3.0"`)

	stdout.Reset()
	runOK(t, projectDir, &stdout, "deploy", "codex")
	assertContains(t, stdout.String(), "deployed java-review to codex")
	assertFileContains(t, filepath.Join(codexDir, "java-review", "SKILL.md"), "Review Java changes")
	assertFileContains(t, filepath.Join(codexDir, "java-review", "references", "rules.md"), "Prefer focused reviews")
}

func TestInstallFromExplicitLocalPath(t *testing.T) {
	workspace := t.TempDir()
	home := filepath.Join(workspace, "skillhub-home")
	projectDir := filepath.Join(workspace, "project")
	localSkill := filepath.Join(workspace, "my-skill")
	t.Setenv("SKILLHUB_HOME", home)

	mustWriteFile(t, filepath.Join(localSkill, "skill.yaml"), strings.TrimSpace(`
name: my-skill
version: 0.1.0
description: Local test skill
entry: SKILL.md
targets:
  - codex
`)+"\n")
	mustWriteFile(t, filepath.Join(localSkill, "SKILL.md"), "# Local Skill\n")

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "init")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "install", localSkill)

	assertContains(t, stdout.String(), "installed my-skill@0.1.0")
	assertFileContains(t, filepath.Join(home, "installed", "my-skill", "skill.yaml"), "Local test skill")
}

func TestRegistryAddGitRecordsMetadataWithoutCloning(t *testing.T) {
	workspace := t.TempDir()
	projectDir := filepath.Join(workspace, "project")
	t.Setenv("SKILLHUB_HOME", filepath.Join(workspace, "skillhub-home"))

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "init")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "registry", "add", "git", "company", "git@gitlab.example.com:ai/skills.git")

	assertContains(t, stdout.String(), "registered company")
	assertFileContains(t, filepath.Join(projectDir, "skillhub.yaml"), `"type": "git"`)
	assertFileContains(t, filepath.Join(projectDir, "skillhub.yaml"), `git@gitlab.example.com:ai/skills.git`)
}

func TestRegistryAddLocalResolvesRelativePathsFromWorkDir(t *testing.T) {
	workspace := t.TempDir()
	projectDir := filepath.Join(workspace, "project")
	registryDir := filepath.Join(projectDir, "skills")
	t.Setenv("SKILLHUB_HOME", filepath.Join(workspace, "skillhub-home"))

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "init")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "registry", "add", "local", "company", "./skills")

	assertFileContains(t, filepath.Join(projectDir, "skillhub.yaml"), registryDir)
}

func runOK(t *testing.T, workDir string, stdout *bytes.Buffer, args ...string) {
	t.Helper()
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
	}
	stderr := &bytes.Buffer{}
	if err := Run(args, stdout, stderr, workDir); err != nil {
		t.Fatalf("Run(%v) failed: %v\nstderr:\n%s", args, err, stderr.String())
	}
}

func mustWriteFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func assertFileContains(t *testing.T, path string, want string) {
	t.Helper()
	assertContains(t, readFile(t, path), want)
}

func assertContains(t *testing.T, got string, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("expected %q to contain %q", got, want)
	}
}
