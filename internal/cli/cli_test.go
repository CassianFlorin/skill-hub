package cli

import (
	"bytes"
	"os"
	"os/exec"
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
	assertContains(t, stdout.String(), "installed platform-team/java-review@1.2.0")
	assertFileContains(t, filepath.Join(home, "installed", "platform-team__java-review", "SKILL.md"), "Review Java changes")
	assertFileContains(t, filepath.Join(home, "skillhub.lock"), `"version": "1.2.0"`)

	stdout.Reset()
	runOK(t, projectDir, &stdout, "list")
	assertContains(t, stdout.String(), "java-review")
	assertContains(t, stdout.String(), "1.2.0")

	mustWriteFile(t, filepath.Join(skillDir, "skill.yaml"), strings.Replace(readFile(t, filepath.Join(skillDir, "skill.yaml")), "version: 1.2.0", "version: 1.3.0", 1))
	stdout.Reset()
	runOK(t, projectDir, &stdout, "update")
	assertContains(t, stdout.String(), "updated platform-team/java-review 1.2.0 -> 1.3.0")
	assertFileContains(t, filepath.Join(home, "skillhub.lock"), `"version": "1.3.0"`)

	stdout.Reset()
	runOK(t, projectDir, &stdout, "deploy", "codex")
	assertContains(t, stdout.String(), "deployed platform-team/java-review to codex")
	assertFileContains(t, filepath.Join(codexDir, "platform-team__java-review", "SKILL.md"), "Review Java changes")
	assertFileContains(t, filepath.Join(codexDir, "platform-team__java-review", "references", "rules.md"), "Prefer focused reviews")
	assertFileContains(t, filepath.Join(home, "skillhub.lock"), `"deployed_runtimes": [`)
}

func TestInstallFromExplicitLocalPath(t *testing.T) {
	workspace := t.TempDir()
	home := filepath.Join(workspace, "skillhub-home")
	projectDir := filepath.Join(workspace, "project")
	localSkill := filepath.Join(workspace, "my-skill")
	t.Setenv("SKILLHUB_HOME", home)

	mustWriteFile(t, filepath.Join(localSkill, "skill.yaml"), strings.TrimSpace(`
name: my-skill
namespace: local
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

	assertContains(t, stdout.String(), "installed local/my-skill@0.1.0")
	assertFileContains(t, filepath.Join(home, "installed", "local__my-skill", "skill.yaml"), "Local test skill")
}

func TestInstallSkillOnlyDirectoryGeneratesMetadataAndNamespaceIdentity(t *testing.T) {
	workspace := t.TempDir()
	home := filepath.Join(workspace, "skillhub-home")
	projectDir := filepath.Join(workspace, "project")
	registryDir := filepath.Join(workspace, "registry")
	skillDir := filepath.Join(registryDir, "obsidian-cli")
	t.Setenv("SKILLHUB_HOME", home)

	mustWriteFile(t, filepath.Join(skillDir, "SKILL.md"), "# Obsidian CLI\n")

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "init")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "registry", "add", "local", "cassian", registryDir)
	stdout.Reset()
	runOK(t, projectDir, &stdout, "install", "cassian/obsidian-cli")

	assertContains(t, stdout.String(), "installed cassian/obsidian-cli@unversioned")
	installedDir := filepath.Join(home, "installed", "cassian__obsidian-cli")
	assertFileContains(t, filepath.Join(installedDir, "SKILL.md"), "# Obsidian CLI")
	assertFileContains(t, filepath.Join(installedDir, "skill.yaml"), "namespace: cassian")
	assertFileContains(t, filepath.Join(installedDir, "skill.yaml"), "generated: true")
	assertFileContains(t, filepath.Join(home, "skillhub.lock"), `"identity": "cassian/obsidian-cli"`)
	assertFileContains(t, filepath.Join(home, "skillhub.lock"), `"version": "unversioned"`)
	assertFileContains(t, filepath.Join(home, "skillhub.lock"), `"checksum": "`)

	stdout.Reset()
	runOK(t, projectDir, &stdout, "list")
	assertContains(t, stdout.String(), "cassian/obsidian-cli")
	assertContains(t, stdout.String(), "unversioned")
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

func TestInstallAndUpdateFromGitRegistry(t *testing.T) {
	workspace := t.TempDir()
	home := filepath.Join(workspace, "skillhub-home")
	projectDir := filepath.Join(workspace, "project")
	remoteWorktree := filepath.Join(workspace, "remote-worktree")
	remoteRepo := filepath.Join(workspace, "skills.git")
	t.Setenv("SKILLHUB_HOME", home)

	writeGitSkill(t, remoteWorktree, "java-review", "1.2.0", "Git Java review skill")
	git(t, remoteWorktree, "init")
	git(t, remoteWorktree, "config", "user.email", "skillhub@example.com")
	git(t, remoteWorktree, "config", "user.name", "SkillHub Test")
	git(t, remoteWorktree, "add", ".")
	git(t, remoteWorktree, "commit", "-m", "initial skill")
	git(t, workspace, "clone", "--bare", remoteWorktree, remoteRepo)

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "init")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "registry", "add", "git", "company", remoteRepo)
	stdout.Reset()
	runOK(t, projectDir, &stdout, "install", "company/java-review")

	assertContains(t, stdout.String(), "installed company/java-review@1.2.0")
	assertFileContains(t, filepath.Join(home, "cache", "registries", "company", "java-review", "SKILL.md"), "Git Java review skill")
	assertFileContains(t, filepath.Join(home, "installed", "company__java-review", "SKILL.md"), "Git Java review skill")
	assertFileContains(t, filepath.Join(home, "skillhub.lock"), `"source_type": "git"`)

	writeGitSkill(t, remoteWorktree, "java-review", "1.3.0", "Git Java review skill v1.3.0")
	git(t, remoteWorktree, "add", ".")
	git(t, remoteWorktree, "commit", "-m", "bump skill")
	git(t, remoteWorktree, "push", remoteRepo, "HEAD")

	stdout.Reset()
	runOK(t, projectDir, &stdout, "update")
	assertContains(t, stdout.String(), "updated company/java-review 1.2.0 -> 1.3.0")
	assertFileContains(t, filepath.Join(home, "installed", "company__java-review", "SKILL.md"), "v1.3.0")
	assertFileContains(t, filepath.Join(home, "skillhub.lock"), `"version": "1.3.0"`)
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

func TestRegistryIndexGenerateWritesCatalog(t *testing.T) {
	workspace := t.TempDir()
	projectDir := filepath.Join(workspace, "project")
	registryDir := filepath.Join(workspace, "registry")
	skillDir := filepath.Join(registryDir, "java-review")
	t.Setenv("SKILLHUB_HOME", filepath.Join(workspace, "skillhub-home"))

	mustWriteFile(t, filepath.Join(skillDir, "skill.yaml"), strings.TrimSpace(`
name: java-review
namespace: platform
version: 1.2.0
description: Java review skill
entry: SKILL.md
targets:
  - codex
  - claude
tags:
  - java
`)+"\n")
	mustWriteFile(t, filepath.Join(skillDir, "SKILL.md"), "# Java Review\n")

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "init")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "registry", "add", "local", "company", registryDir)
	stdout.Reset()
	runOK(t, projectDir, &stdout, "registry", "index", "generate", "company")

	assertContains(t, stdout.String(), "indexed company with 1 skills")
	indexPath := filepath.Join(registryDir, "skillhub.index.json")
	assertFileContains(t, indexPath, `"identity": "platform/java-review"`)
	assertFileContains(t, indexPath, `"version": "1.2.0"`)
	assertFileContains(t, indexPath, `"targets": [`)
	assertFileContains(t, indexPath, `"checksum": "sha256:`)
}

func TestInstallUsesRegistryIndexForIdentityLookup(t *testing.T) {
	workspace := t.TempDir()
	home := filepath.Join(workspace, "skillhub-home")
	projectDir := filepath.Join(workspace, "project")
	registryDir := filepath.Join(workspace, "registry")
	skillDir := filepath.Join(registryDir, "java-review")
	t.Setenv("SKILLHUB_HOME", home)

	mustWriteFile(t, filepath.Join(skillDir, "skill.yaml"), strings.TrimSpace(`
name: java-review
namespace: platform
version: 1.2.0
entry: SKILL.md
`)+"\n")
	mustWriteFile(t, filepath.Join(skillDir, "SKILL.md"), "# Java Review\n")

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "init")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "registry", "add", "local", "company", registryDir)
	stdout.Reset()
	runOK(t, projectDir, &stdout, "registry", "index", "generate", "company")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "install", "company/platform/java-review")

	assertContains(t, stdout.String(), "installed platform/java-review@1.2.0")
	assertFileContains(t, filepath.Join(home, "installed", "platform__java-review", "SKILL.md"), "# Java Review")
}

func TestDeployCodexDryRunDoesNotWriteFiles(t *testing.T) {
	workspace := t.TempDir()
	home := filepath.Join(workspace, "skillhub-home")
	codexDir := filepath.Join(workspace, "codex-skills")
	projectDir := filepath.Join(workspace, "project")
	localSkill := filepath.Join(workspace, "my-skill")
	t.Setenv("SKILLHUB_HOME", home)
	t.Setenv("SKILLHUB_CODEX_DIR", codexDir)

	mustWriteFile(t, filepath.Join(localSkill, "skill.yaml"), strings.TrimSpace(`
name: my-skill
namespace: local
version: 0.1.0
entry: SKILL.md
`)+"\n")
	mustWriteFile(t, filepath.Join(localSkill, "SKILL.md"), "# Local Skill\n")

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "init")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "install", localSkill)
	stdout.Reset()
	runOK(t, projectDir, &stdout, "deploy", "codex", "local/my-skill", "--dry-run")

	assertContains(t, stdout.String(), "would deploy local/my-skill to codex")
	assertPathMissing(t, filepath.Join(codexDir, "local__my-skill", "SKILL.md"))
}

func TestDeployCodexRequiresForceWhenTargetExists(t *testing.T) {
	workspace := t.TempDir()
	home := filepath.Join(workspace, "skillhub-home")
	codexDir := filepath.Join(workspace, "codex-skills")
	projectDir := filepath.Join(workspace, "project")
	localSkill := filepath.Join(workspace, "my-skill")
	t.Setenv("SKILLHUB_HOME", home)
	t.Setenv("SKILLHUB_CODEX_DIR", codexDir)

	mustWriteFile(t, filepath.Join(localSkill, "skill.yaml"), strings.TrimSpace(`
name: my-skill
namespace: local
version: 0.1.0
entry: SKILL.md
`)+"\n")
	mustWriteFile(t, filepath.Join(localSkill, "SKILL.md"), "# Local Skill\n")
	mustWriteFile(t, filepath.Join(codexDir, "local__my-skill", "SKILL.md"), "# Existing\n")

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "init")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "install", localSkill)

	stderr := &bytes.Buffer{}
	err := Run([]string{"deploy", "codex", "local/my-skill"}, &stdout, stderr, projectDir)
	if err == nil {
		t.Fatal("expected deploy conflict")
	}
	assertContains(t, err.Error(), "target already exists")

	stdout.Reset()
	runOK(t, projectDir, &stdout, "deploy", "codex", "local/my-skill", "--force")
	assertContains(t, stdout.String(), "deployed local/my-skill to codex")
	assertFileContains(t, filepath.Join(codexDir, "local__my-skill", "SKILL.md"), "# Local Skill")
}

func TestDeployClaudeCopiesInstalledSkill(t *testing.T) {
	workspace := t.TempDir()
	home := filepath.Join(workspace, "skillhub-home")
	claudeDir := filepath.Join(workspace, "claude-skills")
	projectDir := filepath.Join(workspace, "project")
	localSkill := filepath.Join(workspace, "my-skill")
	t.Setenv("SKILLHUB_HOME", home)
	t.Setenv("SKILLHUB_CLAUDE_DIR", claudeDir)

	mustWriteFile(t, filepath.Join(localSkill, "skill.yaml"), strings.TrimSpace(`
name: my-skill
namespace: local
version: 0.1.0
entry: SKILL.md
targets:
  - claude
`)+"\n")
	mustWriteFile(t, filepath.Join(localSkill, "SKILL.md"), "# Local Skill\n")

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "init")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "install", localSkill)
	stdout.Reset()
	runOK(t, projectDir, &stdout, "deploy", "claude", "local/my-skill")

	assertContains(t, stdout.String(), "deployed local/my-skill to claude")
	assertFileContains(t, filepath.Join(claudeDir, "local__my-skill", "SKILL.md"), "# Local Skill")
	assertFileContains(t, filepath.Join(home, "skillhub.lock"), `"claude"`)
}

func writeGitSkill(t *testing.T, root string, name string, version string, description string) {
	t.Helper()
	skillDir := filepath.Join(root, name)
	mustWriteFile(t, filepath.Join(skillDir, "skill.yaml"), strings.TrimSpace(`
name: `+name+`
version: `+version+`
description: `+description+`
entry: SKILL.md
targets:
  - codex
`)+"\n")
	mustWriteFile(t, filepath.Join(skillDir, "SKILL.md"), "# "+name+"\n\n"+description+"\n")
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir git dir: %v", err)
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(output))
	}
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

func assertPathMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("expected %s to be missing", path)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", path, err)
	}
}

func assertContains(t *testing.T, got string, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("expected %q to contain %q", got, want)
	}
}
