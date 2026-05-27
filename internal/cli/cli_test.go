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
	assertFileContains(t, filepath.Join(projectDir, "skillhub.yaml"), `"hub"`)
	assertFileContains(t, filepath.Join(projectDir, "skillhub.yaml"), `https://github.com/CassianFlorin/skill-hub-registry.git`)

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

func TestInstallFromLocalRegistryRequiresPinnedVersionToMatch(t *testing.T) {
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
	runOK(t, projectDir, &stdout, "install", "company/java-review@1.2.0")
	assertContains(t, stdout.String(), "installed platform/java-review@1.2.0")
	assertFileContains(t, filepath.Join(home, "skillhub.lock"), `"source_ref": "1.2.0"`)

	stderr := &bytes.Buffer{}
	err := Run([]string{"install", "company/java-review@2.0.0"}, &stdout, stderr, projectDir)
	if err == nil {
		t.Fatal("expected version mismatch")
	}
	assertContains(t, err.Error(), "version 2.0.0 not available for platform/java-review")
}

func TestInstallFromGitRegistryPinsTagAndRecordsCommit(t *testing.T) {
	workspace := t.TempDir()
	home := filepath.Join(workspace, "skillhub-home")
	projectDir := filepath.Join(workspace, "project")
	remoteWorktree := filepath.Join(workspace, "remote-worktree")
	remoteRepo := filepath.Join(workspace, "skills.git")
	t.Setenv("SKILLHUB_HOME", home)

	writeGitSkill(t, remoteWorktree, "java-review", "1.2.0", "Git Java review skill v1.2.0")
	git(t, remoteWorktree, "init")
	git(t, remoteWorktree, "config", "user.email", "skillhub@example.com")
	git(t, remoteWorktree, "config", "user.name", "SkillHub Test")
	git(t, remoteWorktree, "add", ".")
	git(t, remoteWorktree, "commit", "-m", "initial skill")
	git(t, remoteWorktree, "tag", "v1.2.0")
	writeGitSkill(t, remoteWorktree, "java-review", "1.3.0", "Git Java review skill v1.3.0")
	git(t, remoteWorktree, "add", ".")
	git(t, remoteWorktree, "commit", "-m", "bump skill")
	git(t, workspace, "clone", "--bare", remoteWorktree, remoteRepo)

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "init")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "registry", "add", "git", "company", remoteRepo)
	stdout.Reset()
	runOK(t, projectDir, &stdout, "install", "company/java-review@v1.2.0")

	assertContains(t, stdout.String(), "installed company/java-review@1.2.0")
	assertFileContains(t, filepath.Join(home, "installed", "company__java-review", "SKILL.md"), "v1.2.0")
	assertFileContains(t, filepath.Join(home, "skillhub.lock"), `"source_ref": "v1.2.0"`)
	assertFileContains(t, filepath.Join(home, "skillhub.lock"), `"source_commit": "`)
}

func TestRollbackRestoresPreviousInstalledVersion(t *testing.T) {
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
	mustWriteFile(t, filepath.Join(skillDir, "SKILL.md"), "# Java Review\n\nv1.2.0\n")

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "init")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "registry", "add", "local", "company", registryDir)
	stdout.Reset()
	runOK(t, projectDir, &stdout, "install", "company/java-review")

	mustWriteFile(t, filepath.Join(skillDir, "skill.yaml"), strings.Replace(readFile(t, filepath.Join(skillDir, "skill.yaml")), "version: 1.2.0", "version: 1.3.0", 1))
	mustWriteFile(t, filepath.Join(skillDir, "SKILL.md"), "# Java Review\n\nv1.3.0\n")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "install", "company/java-review")
	assertFileContains(t, filepath.Join(home, "installed", "platform__java-review", "SKILL.md"), "v1.3.0")

	stdout.Reset()
	runOK(t, projectDir, &stdout, "rollback", "platform/java-review")
	assertContains(t, stdout.String(), "rolled back platform/java-review to 1.2.0")
	assertFileContains(t, filepath.Join(home, "installed", "platform__java-review", "SKILL.md"), "v1.2.0")
	assertFileContains(t, filepath.Join(home, "skillhub.lock"), `"version": "1.2.0"`)
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
	assertFileContains(t, indexPath, `"schema_version": "2"`)
	assertFileContains(t, indexPath, `"source": {`)
	assertFileContains(t, indexPath, `"type": "registry"`)
	assertFileContains(t, indexPath, `"path": "java-review"`)
	assertFileContains(t, indexPath, `"trust": {`)
	assertFileContains(t, indexPath, `"level": "private"`)
	assertFileContains(t, indexPath, `"updated_at": "`)
	assertFileContains(t, indexPath, `"checksum": "sha256:`)
	assertFileNotContains(t, indexPath, `"source_type"`)
	assertFileNotContains(t, indexPath, `"source_path"`)
}

func TestRegistryIndexValidateRejectsOldSchema(t *testing.T) {
	workspace := t.TempDir()
	projectDir := filepath.Join(workspace, "project")
	registryDir := filepath.Join(workspace, "registry")
	t.Setenv("SKILLHUB_HOME", filepath.Join(workspace, "skillhub-home"))

	mustWriteFile(t, filepath.Join(registryDir, "skillhub.index.json"), strings.TrimSpace(`
{
  "registry": "company",
  "generated_at": "2026-05-27T00:00:00Z",
  "skills": []
}
`)+"\n")

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "init")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "registry", "add", "local", "company", registryDir)

	stderr := &bytes.Buffer{}
	err := Run([]string{"registry", "index", "validate", "company"}, &stdout, stderr, projectDir)
	if err == nil {
		t.Fatal("expected old schema validation failure")
	}
	assertContains(t, err.Error(), "unsupported index schema")
}

func TestRegistryIndexValidateRejectsUnknownSourceAndTrust(t *testing.T) {
	workspace := t.TempDir()
	projectDir := filepath.Join(workspace, "project")
	registryDir := filepath.Join(workspace, "registry")
	t.Setenv("SKILLHUB_HOME", filepath.Join(workspace, "skillhub-home"))

	mustWriteFile(t, filepath.Join(registryDir, "skillhub.index.json"), strings.TrimSpace(`
{
  "schema_version": "2",
  "registry": "company",
  "generated_at": "2026-05-27T00:00:00Z",
  "skills": [
    {
      "identity": "platform/java-review",
      "name": "java-review",
      "namespace": "platform",
      "version": "1.2.0",
      "description": "Java review skill",
      "targets": ["codex"],
      "source": {"type": "bogus", "path": "java-review"},
      "trust": {"level": "mystery"},
      "featured": false,
      "updated_at": "2026-05-27"
    }
  ]
}
`)+"\n")

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "init")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "registry", "add", "local", "company", registryDir)

	stderr := &bytes.Buffer{}
	err := Run([]string{"registry", "index", "validate", "company"}, &stdout, stderr, projectDir)
	if err == nil {
		t.Fatal("expected unknown source type validation failure")
	}
	assertContains(t, err.Error(), `unsupported source type "bogus"`)
}

func TestRegistryIndexValidateRejectsUnsupportedTarget(t *testing.T) {
	workspace := t.TempDir()
	projectDir := filepath.Join(workspace, "project")
	registryDir := filepath.Join(workspace, "registry")
	t.Setenv("SKILLHUB_HOME", filepath.Join(workspace, "skillhub-home"))

	mustWriteFile(t, filepath.Join(registryDir, "skillhub.index.json"), strings.TrimSpace(`
{
  "schema_version": "2",
  "registry": "company",
  "generated_at": "2026-05-27T00:00:00Z",
  "skills": [
    {
      "identity": "platform/java-review",
      "name": "java-review",
      "namespace": "platform",
      "version": "1.2.0",
      "description": "Java review skill",
      "targets": ["codex", "unknown-runtime"],
      "source": {"type": "git", "url": "https://example.com/skills.git", "path": "java-review"},
      "trust": {"level": "curated"},
      "featured": false,
      "updated_at": "2026-05-27"
    }
  ]
}
`)+"\n")

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "init")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "registry", "add", "local", "company", registryDir)

	stderr := &bytes.Buffer{}
	err := Run([]string{"registry", "index", "validate", "company"}, &stdout, stderr, projectDir)
	if err == nil {
		t.Fatal("expected unsupported target validation failure")
	}
	assertContains(t, err.Error(), `unsupported target "unknown-runtime"`)
}

func TestRegistryIndexValidateRequiresFeaturedSkillQualityFields(t *testing.T) {
	workspace := t.TempDir()
	projectDir := filepath.Join(workspace, "project")
	registryDir := filepath.Join(workspace, "registry")
	t.Setenv("SKILLHUB_HOME", filepath.Join(workspace, "skillhub-home"))

	mustWriteFile(t, filepath.Join(registryDir, "skillhub.index.json"), strings.TrimSpace(`
{
  "schema_version": "2",
  "registry": "company",
  "generated_at": "2026-05-27T00:00:00Z",
  "skills": [
    {
      "identity": "platform/java-review",
      "name": "java-review",
      "namespace": "platform",
      "version": "1.2.0",
      "description": "Java review skill",
      "targets": ["codex"],
      "source": {"type": "git", "url": "https://example.com/skills.git", "path": "java-review"},
      "trust": {"level": "official"},
      "featured": true,
      "updated_at": "2026-05-27"
    }
  ]
}
`)+"\n")

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "init")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "registry", "add", "local", "company", registryDir)

	stderr := &bytes.Buffer{}
	err := Run([]string{"registry", "index", "validate", "company"}, &stdout, stderr, projectDir)
	if err == nil {
		t.Fatal("expected featured quality validation failure")
	}
	assertContains(t, err.Error(), "featured skill platform/java-review missing tags")
}

func TestRegistryIndexGenerateRejectsMissingCatalogRequiredFields(t *testing.T) {
	workspace := t.TempDir()
	projectDir := filepath.Join(workspace, "project")
	registryDir := filepath.Join(workspace, "registry")
	skillDir := filepath.Join(registryDir, "java-review")
	t.Setenv("SKILLHUB_HOME", filepath.Join(workspace, "skillhub-home"))

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

	stderr := &bytes.Buffer{}
	err := Run([]string{"registry", "index", "generate", "company"}, &stdout, stderr, projectDir)
	if err == nil {
		t.Fatal("expected generate validation failure")
	}
	assertContains(t, err.Error(), "missing description")
	assertPathMissing(t, filepath.Join(registryDir, "skillhub.index.json"))
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
description: Java review skill
entry: SKILL.md
targets:
  - codex
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

func TestSearchFindsSkillsFromRegistryIndexes(t *testing.T) {
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
tags:
  - java
  - review
`)+"\n")
	mustWriteFile(t, filepath.Join(skillDir, "SKILL.md"), "# Java Review\n")

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "init")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "registry", "add", "local", "company", registryDir)
	stdout.Reset()
	runOK(t, projectDir, &stdout, "registry", "index", "generate", "company")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "search", "review")

	assertContains(t, stdout.String(), "company/platform/java-review\t1.2.0\tcodex\tprivate\t-")
	assertContains(t, stdout.String(), "Java review skill")
}

func TestSearchShowsTargetsAndMatchesTarget(t *testing.T) {
	workspace := t.TempDir()
	projectDir := filepath.Join(workspace, "project")
	registryDir := filepath.Join(workspace, "registry")
	t.Setenv("SKILLHUB_HOME", filepath.Join(workspace, "skillhub-home"))

	mustWriteFile(t, filepath.Join(registryDir, "skillhub.index.json"), strings.TrimSpace(`
{
  "schema_version": "2",
  "registry": "company",
  "generated_at": "2026-05-27T00:00:00Z",
  "skills": [
    {
      "identity": "platform/java-review",
      "name": "java-review",
      "namespace": "platform",
      "version": "1.2.0",
      "description": "Java review skill",
      "targets": ["codex", "claude"],
      "tags": ["java", "review"],
      "source": {"type": "git", "url": "https://example.com/skills.git", "path": "java-review"},
      "trust": {"level": "curated"},
      "featured": false,
      "updated_at": "2026-05-27"
    }
  ]
}
`)+"\n")

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "init")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "registry", "add", "local", "company", registryDir)
	stdout.Reset()
	runOK(t, projectDir, &stdout, "search", "claude")

	assertContains(t, stdout.String(), "company/platform/java-review\t1.2.0\tcodex,claude\tcurated\t-")
	assertContains(t, stdout.String(), "Java review skill")
}

func TestInfoShowsRegistryIndexSkillDetails(t *testing.T) {
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
	stdout.Reset()
	runOK(t, projectDir, &stdout, "info", "company/platform/java-review")

	assertContains(t, stdout.String(), "identity: platform/java-review")
	assertContains(t, stdout.String(), "registry: company")
	assertContains(t, stdout.String(), "version: 1.2.0")
	assertContains(t, stdout.String(), "targets: codex, claude")
	assertContains(t, stdout.String(), "tags: java")
	assertContains(t, stdout.String(), "source.type: registry")
	assertContains(t, stdout.String(), "source.path: java-review")
	assertContains(t, stdout.String(), "trust: private")
	assertContains(t, stdout.String(), "install: skillhub install company/platform/java-review")
	assertContains(t, stdout.String(), "checksum: sha256:")
}

func TestInfoShowsReviewDetailsForOfficialSkill(t *testing.T) {
	workspace := t.TempDir()
	projectDir := filepath.Join(workspace, "project")
	registryDir := filepath.Join(workspace, "registry")
	t.Setenv("SKILLHUB_HOME", filepath.Join(workspace, "skillhub-home"))

	mustWriteFile(t, filepath.Join(registryDir, "skillhub.index.json"), strings.TrimSpace(`
{
  "schema_version": "2",
  "registry": "company",
  "generated_at": "2026-05-27T00:00:00Z",
  "skills": [
    {
      "identity": "official/skill-authoring-guide",
      "name": "skill-authoring-guide",
      "namespace": "official",
      "version": "0.1.0",
      "description": "Guide agents to create skill packages.",
      "targets": ["codex", "claude"],
      "tags": ["skill", "authoring"],
      "source": {"type": "git", "url": "https://example.com/skills.git", "path": "skill-authoring-guide"},
      "maintainers": ["CassianFlorin"],
      "license": "MIT",
      "trust": {"level": "official", "reviewed_at": "2026-05-27", "reviewer": "CassianFlorin"},
      "featured": true,
      "updated_at": "2026-05-27"
    }
  ]
}
`)+"\n")

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "init")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "registry", "add", "local", "company", registryDir)
	stdout.Reset()
	runOK(t, projectDir, &stdout, "info", "company/official/skill-authoring-guide")

	assertContains(t, stdout.String(), "trust: official")
	assertContains(t, stdout.String(), "trust.reviewed_at: 2026-05-27")
	assertContains(t, stdout.String(), "trust.reviewer: CassianFlorin")
}

func TestRegistryListShowsConfiguredRegistries(t *testing.T) {
	workspace := t.TempDir()
	projectDir := filepath.Join(workspace, "project")
	registryDir := filepath.Join(workspace, "registry")
	t.Setenv("SKILLHUB_HOME", filepath.Join(workspace, "skillhub-home"))

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "init")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "registry", "add", "local", "company", registryDir)
	stdout.Reset()
	runOK(t, projectDir, &stdout, "registry", "list")

	assertContains(t, stdout.String(), "hub\tgit\thttps://github.com/CassianFlorin/skill-hub-registry.git")
	assertContains(t, stdout.String(), "company\tlocal\t"+registryDir)
}

func TestRegistrySyncValidatesLocalRegistryIndex(t *testing.T) {
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
`)+"\n")
	mustWriteFile(t, filepath.Join(skillDir, "SKILL.md"), "# Java Review\n")

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "init")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "registry", "add", "local", "company", registryDir)
	stdout.Reset()
	runOK(t, projectDir, &stdout, "registry", "index", "generate", "company")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "registry", "sync", "company")

	assertContains(t, stdout.String(), "synced company with 1 skills")
}

func TestRegistrySyncDetectsMissingLocalSourcePath(t *testing.T) {
	workspace := t.TempDir()
	projectDir := filepath.Join(workspace, "project")
	registryDir := filepath.Join(workspace, "registry")
	t.Setenv("SKILLHUB_HOME", filepath.Join(workspace, "skillhub-home"))

	mustWriteFile(t, filepath.Join(registryDir, "skillhub.index.json"), strings.TrimSpace(`
{
  "schema_version": "2",
  "registry": "company",
  "generated_at": "2026-05-27T00:00:00Z",
  "skills": [
    {
      "identity": "platform/java-review",
      "name": "java-review",
      "namespace": "platform",
      "version": "1.2.0",
      "description": "Java review skill",
      "targets": ["codex"],
      "source": {"type": "registry", "path": "missing-skill"},
      "trust": {"level": "private"},
      "featured": false,
      "updated_at": "2026-05-27"
    }
  ]
}
`)+"\n")

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "init")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "registry", "add", "local", "company", registryDir)

	stderr := &bytes.Buffer{}
	err := Run([]string{"registry", "sync", "company"}, &stdout, stderr, projectDir)
	if err == nil {
		t.Fatal("expected sync validation failure")
	}
	assertContains(t, err.Error(), "validate platform/java-review")
}

func TestRegistrySyncRejectsEscapingLocalSourcePath(t *testing.T) {
	workspace := t.TempDir()
	projectDir := filepath.Join(workspace, "project")
	registryDir := filepath.Join(workspace, "registry")
	outsideSkillDir := filepath.Join(workspace, "outside-skill")
	t.Setenv("SKILLHUB_HOME", filepath.Join(workspace, "skillhub-home"))

	mustWriteFile(t, filepath.Join(outsideSkillDir, "skill.yaml"), strings.TrimSpace(`
name: outside-skill
namespace: platform
version: 1.2.0
description: Outside skill
entry: SKILL.md
targets:
  - codex
`)+"\n")
	mustWriteFile(t, filepath.Join(outsideSkillDir, "SKILL.md"), "# Outside Skill\n")
	mustWriteFile(t, filepath.Join(registryDir, "skillhub.index.json"), strings.TrimSpace(`
{
  "schema_version": "2",
  "registry": "company",
  "generated_at": "2026-05-27T00:00:00Z",
  "skills": [
    {
      "identity": "platform/outside-skill",
      "name": "outside-skill",
      "namespace": "platform",
      "version": "1.2.0",
      "description": "Outside skill",
      "targets": ["codex"],
      "source": {"type": "registry", "path": "../outside-skill"},
      "trust": {"level": "private"},
      "featured": false,
      "updated_at": "2026-05-27"
    }
  ]
}
`)+"\n")

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "init")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "registry", "add", "local", "company", registryDir)

	stderr := &bytes.Buffer{}
	err := Run([]string{"registry", "sync", "company"}, &stdout, stderr, projectDir)
	if err == nil {
		t.Fatal("expected escaping source path validation failure")
	}
	assertContains(t, err.Error(), "escapes registry root")
}

func TestInstallRejectsEscapingRegistrySourcePath(t *testing.T) {
	workspace := t.TempDir()
	projectDir := filepath.Join(workspace, "project")
	registryDir := filepath.Join(workspace, "registry")
	outsideSkillDir := filepath.Join(workspace, "outside-skill")
	t.Setenv("SKILLHUB_HOME", filepath.Join(workspace, "skillhub-home"))

	mustWriteFile(t, filepath.Join(outsideSkillDir, "skill.yaml"), strings.TrimSpace(`
name: outside-skill
namespace: platform
version: 1.2.0
description: Outside skill
entry: SKILL.md
targets:
  - codex
`)+"\n")
	mustWriteFile(t, filepath.Join(outsideSkillDir, "SKILL.md"), "# Outside Skill\n")
	mustWriteFile(t, filepath.Join(registryDir, "skillhub.index.json"), strings.TrimSpace(`
{
  "schema_version": "2",
  "registry": "company",
  "generated_at": "2026-05-27T00:00:00Z",
  "skills": [
    {
      "identity": "platform/outside-skill",
      "name": "outside-skill",
      "namespace": "platform",
      "version": "1.2.0",
      "description": "Outside skill",
      "targets": ["codex"],
      "source": {"type": "registry", "path": "../outside-skill"},
      "trust": {"level": "private"},
      "featured": false,
      "updated_at": "2026-05-27"
    }
  ]
}
`)+"\n")

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "init")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "registry", "add", "local", "company", registryDir)

	stderr := &bytes.Buffer{}
	err := Run([]string{"install", "company/platform/outside-skill"}, &stdout, stderr, projectDir)
	if err == nil {
		t.Fatal("expected escaping source path install failure")
	}
	assertContains(t, err.Error(), "escapes registry root")
}

func TestInstallRejectsEscapingExternalGitSourcePath(t *testing.T) {
	workspace := t.TempDir()
	projectDir := filepath.Join(workspace, "project")
	registryDir := filepath.Join(workspace, "registry")
	remoteWorktree := filepath.Join(workspace, "remote-worktree")
	remoteRepo := filepath.Join(workspace, "skills.git")
	t.Setenv("SKILLHUB_HOME", filepath.Join(workspace, "skillhub-home"))

	writeGitSkill(t, filepath.Join(remoteWorktree, "safe"), "outside-skill", "1.2.0", "Outside skill")
	git(t, remoteWorktree, "init")
	git(t, remoteWorktree, "config", "user.email", "skillhub@example.com")
	git(t, remoteWorktree, "config", "user.name", "SkillHub Test")
	git(t, remoteWorktree, "add", ".")
	git(t, remoteWorktree, "commit", "-m", "initial skill")
	git(t, workspace, "clone", "--bare", remoteWorktree, remoteRepo)
	mustWriteFile(t, filepath.Join(registryDir, "skillhub.index.json"), strings.TrimSpace(`
{
  "schema_version": "2",
  "registry": "company",
  "generated_at": "2026-05-27T00:00:00Z",
  "skills": [
    {
      "identity": "safe/outside-skill",
      "name": "outside-skill",
      "namespace": "safe",
      "version": "1.2.0",
      "description": "Outside skill",
      "targets": ["codex"],
      "source": {"type": "git", "url": "`+remoteRepo+`", "path": "../safe/outside-skill"},
      "trust": {"level": "curated"},
      "featured": false,
      "updated_at": "2026-05-27"
    }
  ]
}
`)+"\n")

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "init")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "registry", "add", "local", "company", registryDir)

	stderr := &bytes.Buffer{}
	err := Run([]string{"install", "company/safe/outside-skill"}, &stdout, stderr, projectDir)
	if err == nil {
		t.Fatal("expected escaping external git source path install failure")
	}
	assertContains(t, err.Error(), "escapes registry root")
}

func TestCatalogListAndFeaturedShowSyncedSkills(t *testing.T) {
	workspace := t.TempDir()
	projectDir := filepath.Join(workspace, "project")
	registryDir := filepath.Join(workspace, "registry")
	t.Setenv("SKILLHUB_HOME", filepath.Join(workspace, "skillhub-home"))

	mustWriteFile(t, filepath.Join(registryDir, "skillhub.index.json"), strings.TrimSpace(`
{
  "schema_version": "2",
  "registry": "company",
  "generated_at": "2026-05-27T00:00:00Z",
  "skills": [
    {
      "identity": "platform/java-review",
      "name": "java-review",
      "namespace": "platform",
      "version": "1.2.0",
      "description": "Java review skill",
      "targets": ["codex", "claude"],
      "tags": ["java", "review"],
      "source": {"type": "git", "url": "https://example.com/skills.git", "path": "java-review", "ref": "v1.2.0"},
      "maintainers": ["platform-team"],
      "license": "MIT",
      "trust": {"level": "curated", "reviewed_at": "2026-05-27"},
      "featured": true,
      "updated_at": "2026-05-27"
    }
  ]
}
`)+"\n")

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "init")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "registry", "add", "local", "company", registryDir)
	stdout.Reset()
	runOK(t, projectDir, &stdout, "catalog", "list", "--target", "codex", "--tag", "java")
	assertContains(t, stdout.String(), "company/platform/java-review")
	assertContains(t, stdout.String(), "curated")
	assertContains(t, stdout.String(), "featured")

	stdout.Reset()
	runOK(t, projectDir, &stdout, "catalog", "featured")
	assertContains(t, stdout.String(), "company/platform/java-review")
}

func TestCatalogListSupportsFeaturedAndOfficialFilters(t *testing.T) {
	workspace := t.TempDir()
	projectDir := filepath.Join(workspace, "project")
	registryDir := filepath.Join(workspace, "registry")
	t.Setenv("SKILLHUB_HOME", filepath.Join(workspace, "skillhub-home"))

	mustWriteFile(t, filepath.Join(registryDir, "skillhub.index.json"), strings.TrimSpace(`
{
  "schema_version": "2",
  "registry": "company",
  "generated_at": "2026-05-27T00:00:00Z",
  "skills": [
    {
      "identity": "official/skill-authoring-guide",
      "name": "skill-authoring-guide",
      "namespace": "official",
      "version": "0.1.0",
      "description": "Guide agents to create skill packages.",
      "targets": ["codex", "claude"],
      "tags": ["skill", "authoring"],
      "source": {"type": "git", "url": "https://example.com/skills.git", "path": "skill-authoring-guide"},
      "trust": {"level": "official", "reviewed_at": "2026-05-27", "reviewer": "CassianFlorin"},
      "featured": true,
      "updated_at": "2026-05-27"
    },
    {
      "identity": "community/git-helper",
      "name": "git-helper",
      "namespace": "community",
      "version": "0.1.0",
      "description": "Community git helper.",
      "targets": ["codex"],
      "tags": ["git"],
      "source": {"type": "git", "url": "https://example.com/skills.git", "path": "git-helper"},
      "trust": {"level": "community"},
      "featured": true,
      "updated_at": "2026-05-27"
    }
  ]
}
`)+"\n")

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "init")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "registry", "add", "local", "company", registryDir)
	stdout.Reset()
	runOK(t, projectDir, &stdout, "catalog", "list", "--featured", "--official")

	assertContains(t, stdout.String(), "company/official/skill-authoring-guide\t0.1.0\tcodex,claude\tofficial\tfeatured")
	assertNotContains(t, stdout.String(), "community/git-helper")
}

func TestInstallFromCatalogExternalGitSource(t *testing.T) {
	workspace := t.TempDir()
	home := filepath.Join(workspace, "skillhub-home")
	projectDir := filepath.Join(workspace, "project")
	registryDir := filepath.Join(workspace, "registry")
	remoteWorktree := filepath.Join(workspace, "remote-worktree")
	remoteRepo := filepath.Join(workspace, "skills.git")
	t.Setenv("SKILLHUB_HOME", home)

	writeGitSkill(t, remoteWorktree, "java-review", "1.2.0", "External Java review skill")
	git(t, remoteWorktree, "init")
	git(t, remoteWorktree, "config", "user.email", "skillhub@example.com")
	git(t, remoteWorktree, "config", "user.name", "SkillHub Test")
	git(t, remoteWorktree, "add", ".")
	git(t, remoteWorktree, "commit", "-m", "initial skill")
	git(t, remoteWorktree, "tag", "v1.2.0")
	git(t, workspace, "clone", "--bare", remoteWorktree, remoteRepo)

	mustWriteFile(t, filepath.Join(registryDir, "skillhub.index.json"), strings.TrimSpace(`
{
  "schema_version": "2",
  "registry": "company",
  "generated_at": "2026-05-27T00:00:00Z",
  "skills": [
    {
      "identity": "company/java-review",
      "name": "java-review",
      "namespace": "company",
      "version": "1.2.0",
      "description": "External Java review skill",
      "targets": ["codex"],
      "tags": ["java"],
      "source": {"type": "git", "url": "`+remoteRepo+`", "path": "java-review", "ref": "v1.2.0"},
      "trust": {"level": "curated"},
      "featured": true,
      "updated_at": "2026-05-27"
    }
  ]
}
`)+"\n")

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "init")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "registry", "add", "local", "company", registryDir)
	stdout.Reset()
	runOK(t, projectDir, &stdout, "install", "company/company/java-review")

	assertContains(t, stdout.String(), "installed company/java-review@1.2.0")
	assertFileContains(t, filepath.Join(home, "installed", "company__java-review", "SKILL.md"), "External Java review skill")
	assertFileContains(t, filepath.Join(home, "skillhub.lock"), `"source_type": "git"`)
	assertFileContains(t, filepath.Join(home, "skillhub.lock"), `"source_url": "`+remoteRepo+`"`)
	assertFileContains(t, filepath.Join(home, "skillhub.lock"), `"source_ref": "v1.2.0"`)
}

func TestInstallFromIndexedGitRegistryHonorsPinnedRef(t *testing.T) {
	workspace := t.TempDir()
	home := filepath.Join(workspace, "skillhub-home")
	projectDir := filepath.Join(workspace, "project")
	remoteWorktree := filepath.Join(workspace, "remote-worktree")
	remoteRepo := filepath.Join(workspace, "skills.git")
	t.Setenv("SKILLHUB_HOME", home)

	writeGitSkill(t, remoteWorktree, "java-review", "1.2.0", "Git Java review skill v1.2.0")
	mustWriteIndex(t, remoteWorktree, "company", "company/java-review", "java-review", "1.2.0", "Git Java review skill v1.2.0")
	git(t, remoteWorktree, "init")
	git(t, remoteWorktree, "config", "user.email", "skillhub@example.com")
	git(t, remoteWorktree, "config", "user.name", "SkillHub Test")
	git(t, remoteWorktree, "add", ".")
	git(t, remoteWorktree, "commit", "-m", "initial skill")
	git(t, remoteWorktree, "tag", "v1.2.0")
	writeGitSkill(t, remoteWorktree, "java-review", "1.3.0", "Git Java review skill v1.3.0")
	mustWriteIndex(t, remoteWorktree, "company", "company/java-review", "java-review", "1.3.0", "Git Java review skill v1.3.0")
	git(t, remoteWorktree, "add", ".")
	git(t, remoteWorktree, "commit", "-m", "bump skill")
	git(t, workspace, "clone", "--bare", remoteWorktree, remoteRepo)

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "init")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "registry", "add", "git", "company", remoteRepo)
	stdout.Reset()
	runOK(t, projectDir, &stdout, "install", "company/java-review@v1.2.0")

	assertContains(t, stdout.String(), "installed company/java-review@1.2.0")
	assertFileContains(t, filepath.Join(home, "installed", "company__java-review", "SKILL.md"), "v1.2.0")
	assertFileContains(t, filepath.Join(home, "skillhub.lock"), `"source_type": "git"`)
	assertFileContains(t, filepath.Join(home, "skillhub.lock"), `"source_ref": "v1.2.0"`)
}

func TestUpdateCatalogExternalGitSourceUsesRecordedSubpathAndRef(t *testing.T) {
	workspace := t.TempDir()
	home := filepath.Join(workspace, "skillhub-home")
	projectDir := filepath.Join(workspace, "project")
	registryDir := filepath.Join(workspace, "registry")
	remoteWorktree := filepath.Join(workspace, "remote-worktree")
	remoteRepo := filepath.Join(workspace, "skills.git")
	t.Setenv("SKILLHUB_HOME", home)

	writeNestedGitSkill(t, remoteWorktree, filepath.Join("nested", "java-review"), "java-review", "1.2.0", "External Java review skill v1.2.0")
	git(t, remoteWorktree, "init")
	git(t, remoteWorktree, "config", "user.email", "skillhub@example.com")
	git(t, remoteWorktree, "config", "user.name", "SkillHub Test")
	git(t, remoteWorktree, "add", ".")
	git(t, remoteWorktree, "commit", "-m", "initial skill")
	git(t, remoteWorktree, "tag", "v1.2.0")
	git(t, workspace, "clone", "--bare", remoteWorktree, remoteRepo)
	writeNestedGitSkill(t, remoteWorktree, filepath.Join("nested", "java-review"), "java-review", "1.3.0", "External Java review skill v1.3.0")
	git(t, remoteWorktree, "add", ".")
	git(t, remoteWorktree, "commit", "-m", "bump skill")
	git(t, remoteWorktree, "push", remoteRepo, "HEAD")

	mustWriteFile(t, filepath.Join(registryDir, "skillhub.index.json"), strings.TrimSpace(`
{
  "schema_version": "2",
  "registry": "company",
  "generated_at": "2026-05-27T00:00:00Z",
  "skills": [
    {
      "identity": "company/java-review",
      "name": "java-review",
      "namespace": "company",
      "version": "1.2.0",
      "description": "External Java review skill",
      "targets": ["codex"],
      "tags": ["java"],
      "source": {"type": "git", "url": "`+remoteRepo+`", "path": "nested/java-review", "ref": "v1.2.0"},
      "trust": {"level": "curated"},
      "featured": true,
      "updated_at": "2026-05-27"
    }
  ]
}
`)+"\n")

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "init")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "registry", "add", "local", "company", registryDir)
	stdout.Reset()
	runOK(t, projectDir, &stdout, "install", "company/company/java-review")
	assertFileContains(t, filepath.Join(home, "installed", "company__java-review", "SKILL.md"), "v1.2.0")
	lockPath := filepath.Join(home, "skillhub.lock")
	mustWriteFile(t, lockPath, withoutLinesContaining(readFile(t, lockPath), "source_subpath", "source_cache"))

	stdout.Reset()
	runOK(t, projectDir, &stdout, "update")

	assertContains(t, stdout.String(), "all skills are current")
	assertFileContains(t, filepath.Join(home, "installed", "company__java-review", "SKILL.md"), "v1.2.0")
	assertFileContains(t, filepath.Join(home, "skillhub.lock"), `"source_subpath": "nested/java-review"`)
}

func TestRegistryIndexValidateDetectsChecksumDrift(t *testing.T) {
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
`)+"\n")
	mustWriteFile(t, filepath.Join(skillDir, "SKILL.md"), "# Java Review\n")

	var stdout bytes.Buffer
	runOK(t, projectDir, &stdout, "init")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "registry", "add", "local", "company", registryDir)
	stdout.Reset()
	runOK(t, projectDir, &stdout, "registry", "index", "generate", "company")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "registry", "index", "validate", "company")
	assertContains(t, stdout.String(), "validated company with 1 skills")

	mustWriteFile(t, filepath.Join(skillDir, "SKILL.md"), "# Java Review\n\nchanged\n")
	stderr := &bytes.Buffer{}
	err := Run([]string{"registry", "index", "validate", "company"}, &stdout, stderr, projectDir)
	if err == nil {
		t.Fatal("expected checksum validation failure")
	}
	assertContains(t, err.Error(), "checksum mismatch for platform/java-review")

	stdout.Reset()
	err = Run([]string{"registry", "sync", "company"}, &stdout, stderr, projectDir)
	if err == nil {
		t.Fatal("expected sync checksum validation failure")
	}
	assertContains(t, err.Error(), "checksum mismatch for platform/java-review")
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

func TestDeployStatusReportsRuntimeState(t *testing.T) {
	workspace := t.TempDir()
	home := filepath.Join(workspace, "skillhub-home")
	codexDir := filepath.Join(workspace, "codex-skills")
	claudeDir := filepath.Join(workspace, "claude-skills")
	projectDir := filepath.Join(workspace, "project")
	localSkill := filepath.Join(workspace, "my-skill")
	t.Setenv("SKILLHUB_HOME", home)
	t.Setenv("SKILLHUB_CODEX_DIR", codexDir)
	t.Setenv("SKILLHUB_CLAUDE_DIR", claudeDir)

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
	runOK(t, projectDir, &stdout, "deploy", "codex", "local/my-skill", "--force")

	stdout.Reset()
	runOK(t, projectDir, &stdout, "deploy", "status")
	assertContains(t, stdout.String(), "local/my-skill\tcodex\tdeployed")
	assertContains(t, stdout.String(), "local/my-skill\tclaude\tmissing")

	mustWriteFile(t, filepath.Join(codexDir, "local__my-skill", "SKILL.md"), "# Drifted\n")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "deploy", "status", "codex")
	assertContains(t, stdout.String(), "local/my-skill\tcodex\tdrifted")
}

func TestUninstallRemovesInstalledSkillButKeepsDeployedCopyByDefault(t *testing.T) {
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
	runOK(t, projectDir, &stdout, "deploy", "codex", "local/my-skill", "--force")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "uninstall", "local/my-skill")

	assertContains(t, stdout.String(), "uninstalled local/my-skill")
	assertPathMissing(t, filepath.Join(home, "installed", "local__my-skill"))
	assertFileContains(t, filepath.Join(codexDir, "local__my-skill", "SKILL.md"), "# Local Skill")
	assertFileNotContains(t, filepath.Join(home, "skillhub.lock"), `"identity": "local/my-skill"`)
}

func TestUninstallWithDeployedRemovesRuntimeCopies(t *testing.T) {
	workspace := t.TempDir()
	home := filepath.Join(workspace, "skillhub-home")
	codexDir := filepath.Join(workspace, "codex-skills")
	claudeDir := filepath.Join(workspace, "claude-skills")
	projectDir := filepath.Join(workspace, "project")
	localSkill := filepath.Join(workspace, "my-skill")
	t.Setenv("SKILLHUB_HOME", home)
	t.Setenv("SKILLHUB_CODEX_DIR", codexDir)
	t.Setenv("SKILLHUB_CLAUDE_DIR", claudeDir)

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
	runOK(t, projectDir, &stdout, "deploy", "codex", "local/my-skill", "--force")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "deploy", "claude", "local/my-skill", "--force")
	stdout.Reset()
	runOK(t, projectDir, &stdout, "uninstall", "local/my-skill", "--deployed")

	assertContains(t, stdout.String(), "uninstalled local/my-skill")
	assertPathMissing(t, filepath.Join(home, "installed", "local__my-skill"))
	assertPathMissing(t, filepath.Join(codexDir, "local__my-skill"))
	assertPathMissing(t, filepath.Join(claudeDir, "local__my-skill"))
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

func writeNestedGitSkill(t *testing.T, root string, subpath string, name string, version string, description string) {
	t.Helper()
	skillDir := filepath.Join(root, subpath)
	mustWriteFile(t, filepath.Join(skillDir, "skill.yaml"), strings.TrimSpace(`
name: `+name+`
namespace: company
version: `+version+`
description: `+description+`
entry: SKILL.md
targets:
  - codex
`)+"\n")
	mustWriteFile(t, filepath.Join(skillDir, "SKILL.md"), "# "+name+"\n\n"+description+"\n")
}

func mustWriteIndex(t *testing.T, root string, registry string, identity string, sourcePath string, version string, description string) {
	t.Helper()
	mustWriteFile(t, filepath.Join(root, "skillhub.index.json"), strings.TrimSpace(`
{
  "schema_version": "2",
  "registry": "`+registry+`",
  "generated_at": "2026-05-27T00:00:00Z",
  "skills": [
    {
      "identity": "`+identity+`",
      "name": "java-review",
      "namespace": "company",
      "version": "`+version+`",
      "description": "`+description+`",
      "targets": ["codex"],
      "tags": ["java"],
      "source": {"type": "registry", "path": "`+sourcePath+`"},
      "trust": {"level": "private"},
      "featured": false,
      "updated_at": "2026-05-27"
    }
  ]
}
`)+"\n")
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

func assertFileNotContains(t *testing.T, path string, want string) {
	t.Helper()
	got := readFile(t, path)
	if strings.Contains(got, want) {
		t.Fatalf("expected %q not to contain %q", got, want)
	}
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

func assertNotContains(t *testing.T, got string, want string) {
	t.Helper()
	if strings.Contains(got, want) {
		t.Fatalf("expected %q not to contain %q", got, want)
	}
}

func withoutLinesContaining(content string, markers ...string) string {
	var kept []string
	for _, line := range strings.Split(content, "\n") {
		remove := false
		for _, marker := range markers {
			if strings.Contains(line, marker) {
				remove = true
				break
			}
		}
		if !remove {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
}
