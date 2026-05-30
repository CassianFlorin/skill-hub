package manage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSummaryCombinesGlobalProjectAndDeploymentState(t *testing.T) {
	workspace := t.TempDir()
	home := filepath.Join(workspace, "home")
	t.Setenv("SKILLHUB_HOME", home)

	writeFile(t, filepath.Join(home, "skillhub.lock"), `{
  "skills": [
    {
      "identity": "official/git-commit-cn",
      "name": "git-commit-cn",
      "namespace": "official",
      "version": "0.1.0",
      "source_type": "registry",
      "source_registry": "hub",
      "source_path": "/registry/git-commit-cn",
      "installed_path": "/home/installed/official__git-commit-cn",
      "checksum": "abc123",
      "targets": ["codex"],
      "updated_at": "2026-05-30T00:00:00Z"
    }
  ]
}`)
	writeFile(t, filepath.Join(workspace, ".codex", "skills", "commerce-data-fix-sql", "SKILL.md"), "# Commerce Data Fix SQL\n")

	summary, err := Summary(workspace)
	if err != nil {
		t.Fatalf("Summary returned error: %v", err)
	}

	assertEqual(t, len(summary.Skills), 2, "skill count")
	assertEqual(t, summary.Skills[0].Scope, ScopeGlobal, "first scope")
	assertEqual(t, summary.Skills[0].Identity, "official/git-commit-cn", "first identity")
	assertEqual(t, summary.Skills[0].RuntimeStates["codex"], "missing", "codex state")
	assertEqual(t, summary.Skills[0].RuntimeStates["claude"], "unsupported", "claude state")
	assertEqual(t, summary.Skills[1].Scope, ScopeProject, "second scope")
	assertEqual(t, summary.Skills[1].Identity, "project/commerce-data-fix-sql", "second identity")
	assertEqual(t, summary.Skills[1].Path, ".codex/skills/commerce-data-fix-sql", "project path")
}

func TestCatalogSearchUsesConfiguredRegistries(t *testing.T) {
	workspace := t.TempDir()
	home := filepath.Join(workspace, "home")
	registryRoot := filepath.Join(workspace, "registry")
	t.Setenv("SKILLHUB_HOME", home)

	writeFile(t, filepath.Join(workspace, "skillhub.yaml"), `{
  "install_dir": "`+filepath.ToSlash(filepath.Join(home, "installed"))+`",
  "registries": {
    "company": {
      "type": "local",
      "path": "`+filepath.ToSlash(registryRoot)+`"
    }
  }
}`)
	writeFile(t, filepath.Join(registryRoot, "skillhub.index.json"), `{
  "schema_version": "2",
  "registry": "company",
  "generated_at": "2026-05-30T00:00:00Z",
  "skills": [
    {
      "identity": "platform/java-review",
      "name": "java-review",
      "namespace": "platform",
      "version": "1.2.0",
      "description": "Review Java services",
      "targets": ["codex"],
      "tags": ["java", "review"],
      "source": {"type": "registry", "path": "java-review"},
      "trust": {"level": "private"},
      "featured": true,
      "updated_at": "2026-05-30"
    }
  ]
}`)

	results, err := SearchCatalog(workspace, "java")
	if err != nil {
		t.Fatalf("SearchCatalog returned error: %v", err)
	}

	assertEqual(t, len(results), 1, "result count")
	assertEqual(t, results[0].Registry, "company", "registry")
	assertEqual(t, results[0].Identity, "platform/java-review", "identity")
	assertEqual(t, results[0].Trust, "private", "trust")
	assertEqual(t, results[0].Featured, true, "featured")
}

func TestOperationConfirmationPolicy(t *testing.T) {
	direct := []OperationRequest{
		{Kind: OperationInstall},
		{Kind: OperationRegistrySync},
		{Kind: OperationUpdate},
		{Kind: OperationDeploy},
	}
	for _, request := range direct {
		if RequiresConfirmation(request) {
			t.Fatalf("%s should not require confirmation", request.Kind)
		}
	}

	dangerous := []OperationRequest{
		{Kind: OperationUninstall},
		{Kind: OperationRollback},
		{Kind: OperationDeploy, Force: true},
		{Kind: OperationRegistryDelete},
		{Kind: OperationProjectOverwrite},
	}
	for _, request := range dangerous {
		if !RequiresConfirmation(request) {
			t.Fatalf("%s should require confirmation", request.Kind)
		}
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertEqual[T comparable](t *testing.T, got T, want T, label string) {
	t.Helper()
	if got != want {
		gotJSON, _ := json.Marshal(got)
		wantJSON, _ := json.Marshal(want)
		t.Fatalf("%s: got %s, want %s", label, gotJSON, wantJSON)
	}
}
