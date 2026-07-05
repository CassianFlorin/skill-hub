package publish

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CassianFlorin/skill-hub/internal/config"
	"github.com/CassianFlorin/skill-hub/internal/registry"
)

func writeSkill(t *testing.T, dir string, name string, version string, body string) string {
	t.Helper()
	skillDir := filepath.Join(dir, name)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir skill: %v", err)
	}
	yaml := strings.Join([]string{
		"name: " + name,
		"namespace: acme",
		"version: " + version,
		"description: Test skill for publish",
		"author: tester",
		"targets:",
		"- codex",
		"- claude",
		"tags:",
		"- review",
	}, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(skillDir, "skill.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatalf("write skill.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	return skillDir
}

func setupLocalRegistry(t *testing.T) (string, string) {
	t.Helper()
	t.Setenv("SKILLHUB_HOME", t.TempDir())
	workDir := t.TempDir()
	registryDir := t.TempDir()
	cfg, err := config.Load(workDir)
	if err != nil {
		t.Fatalf("config load: %v", err)
	}
	cfg.Registries["company"] = config.Registry{Type: "local", Path: registryDir}
	if err := config.Save(workDir, cfg); err != nil {
		t.Fatalf("config save: %v", err)
	}
	return workDir, registryDir
}

func TestPublishLocalCreatesEntry(t *testing.T) {
	workDir, registryDir := setupLocalRegistry(t)
	skillDir := writeSkill(t, t.TempDir(), "review", "1.0.0", "# review\n")

	result, err := Publish(workDir, skillDir, Options{Registry: "company"})
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if result.Action != ActionCreated {
		t.Errorf("Action = %q, want created", result.Action)
	}
	if result.Identity != "acme/review" {
		t.Errorf("Identity = %q", result.Identity)
	}
	if result.Dest != "review" {
		t.Errorf("Dest = %q, want review", result.Dest)
	}
	index, err := registry.LoadIndex(registryDir)
	if err != nil {
		t.Fatalf("LoadIndex: %v", err)
	}
	if len(index.Skills) != 1 || index.Skills[0].Identity != "acme/review" || index.Skills[0].Version != "1.0.0" {
		t.Fatalf("index skills = %+v", index.Skills)
	}
	if index.Skills[0].Trust.Level != registry.TrustPrivate {
		t.Errorf("trust = %q, want private", index.Skills[0].Trust.Level)
	}
	if index.Skills[0].Checksum == "" {
		t.Error("checksum missing in index entry")
	}
	if _, err := os.Stat(filepath.Join(registryDir, "review", "SKILL.md")); err != nil {
		t.Errorf("published files missing: %v", err)
	}
	if _, err := registry.ValidateLoadedIndex(registryDir, "company"); err != nil {
		t.Errorf("published registry failed validation: %v", err)
	}
}

func TestPublishLocalUpdatePreservesTrustAndFeatured(t *testing.T) {
	workDir, registryDir := setupLocalRegistry(t)
	skillDir := writeSkill(t, t.TempDir(), "review", "1.0.0", "# review\n")
	if _, err := Publish(workDir, skillDir, Options{Registry: "company"}); err != nil {
		t.Fatalf("first publish: %v", err)
	}
	index, err := registry.LoadIndex(registryDir)
	if err != nil {
		t.Fatalf("LoadIndex: %v", err)
	}
	index.Skills[0].Trust = registry.IndexTrust{Level: registry.TrustCurated, Reviewer: "boss"}
	index.Skills[0].Featured = true
	index.Skills[0].Tags = []string{"review"}
	index.Skills[0].License = "MIT"
	if err := registry.WriteIndex(registryDir, index); err != nil {
		t.Fatalf("WriteIndex: %v", err)
	}

	skillDir = writeSkill(t, t.TempDir(), "review", "1.1.0", "# review v2\n")
	result, err := Publish(workDir, skillDir, Options{Registry: "company"})
	if err != nil {
		t.Fatalf("second publish: %v", err)
	}
	if result.Action != ActionUpdated {
		t.Errorf("Action = %q, want updated", result.Action)
	}
	index, err = registry.LoadIndex(registryDir)
	if err != nil {
		t.Fatalf("LoadIndex after update: %v", err)
	}
	entry := index.Skills[0]
	if entry.Version != "1.1.0" {
		t.Errorf("version = %q, want 1.1.0", entry.Version)
	}
	if entry.Trust.Level != registry.TrustCurated || entry.Trust.Reviewer != "boss" {
		t.Errorf("trust not preserved: %+v", entry.Trust)
	}
	if !entry.Featured {
		t.Error("featured not preserved")
	}
	if entry.License != "MIT" {
		t.Errorf("license not preserved: %q", entry.License)
	}
}

func TestPublishSameVersionSameContentUnchanged(t *testing.T) {
	workDir, _ := setupLocalRegistry(t)
	skillDir := writeSkill(t, t.TempDir(), "review", "1.0.0", "# review\n")
	if _, err := Publish(workDir, skillDir, Options{Registry: "company"}); err != nil {
		t.Fatalf("first publish: %v", err)
	}
	result, err := Publish(workDir, skillDir, Options{Registry: "company"})
	if err != nil {
		t.Fatalf("second publish: %v", err)
	}
	if result.Action != ActionUnchanged {
		t.Errorf("Action = %q, want unchanged", result.Action)
	}
}

func TestPublishSameVersionDifferentContentFails(t *testing.T) {
	workDir, _ := setupLocalRegistry(t)
	skillDir := writeSkill(t, t.TempDir(), "review", "1.0.0", "# review\n")
	if _, err := Publish(workDir, skillDir, Options{Registry: "company"}); err != nil {
		t.Fatalf("first publish: %v", err)
	}
	skillDir = writeSkill(t, t.TempDir(), "review", "1.0.0", "# review changed\n")
	if _, err := Publish(workDir, skillDir, Options{Registry: "company"}); err == nil || !strings.Contains(err.Error(), "bump the version") {
		t.Errorf("expected version conflict error, got %v", err)
	}
}

func TestPublishLowerVersionFails(t *testing.T) {
	workDir, _ := setupLocalRegistry(t)
	skillDir := writeSkill(t, t.TempDir(), "review", "1.2.0", "# review\n")
	if _, err := Publish(workDir, skillDir, Options{Registry: "company"}); err != nil {
		t.Fatalf("first publish: %v", err)
	}
	skillDir = writeSkill(t, t.TempDir(), "review", "1.1.9", "# older\n")
	if _, err := Publish(workDir, skillDir, Options{Registry: "company"}); err == nil || !strings.Contains(err.Error(), "lower than published") {
		t.Errorf("expected downgrade error, got %v", err)
	}
}

func TestPublishDryRunWritesNothing(t *testing.T) {
	workDir, registryDir := setupLocalRegistry(t)
	skillDir := writeSkill(t, t.TempDir(), "review", "1.0.0", "# review\n")
	result, err := Publish(workDir, skillDir, Options{Registry: "company", DryRun: true})
	if err != nil {
		t.Fatalf("Publish dry-run: %v", err)
	}
	if !result.DryRun || result.Action != ActionCreated {
		t.Errorf("unexpected result: %+v", result)
	}
	if result.NewEntry.Identity != "acme/review" {
		t.Errorf("NewEntry.Identity = %q", result.NewEntry.Identity)
	}
	if _, err := os.Stat(filepath.Join(registryDir, registry.IndexFileName)); !os.IsNotExist(err) {
		t.Errorf("dry-run wrote index: %v", err)
	}
	if _, err := os.Stat(filepath.Join(registryDir, "review")); !os.IsNotExist(err) {
		t.Errorf("dry-run copied files: %v", err)
	}
}

func TestPublishValidationFailures(t *testing.T) {
	workDir, _ := setupLocalRegistry(t)
	base := t.TempDir()
	skillDir := filepath.Join(base, "bad")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# bad\n"), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	cases := []struct {
		name string
		yaml string
		want string
	}{
		{"missing skill.yaml", "", "requires skill.yaml"},
		{"missing version", "name: bad\nnamespace: acme\ndescription: d\ntargets:\n- codex\n", "explicit version"},
		{"missing description", "name: bad\nnamespace: acme\nversion: 1.0.0\ntargets:\n- codex\n", "description"},
		{"missing targets", "name: bad\nnamespace: acme\nversion: 1.0.0\ndescription: d\n", "target"},
		{"bad target", "name: bad\nnamespace: acme\nversion: 1.0.0\ndescription: d\ntargets:\n- vim\n", "not supported"},
		{"missing namespace", "name: bad\nversion: 1.0.0\ndescription: d\ntargets:\n- codex\n", "namespace or author"},
	}
	for _, testCase := range cases {
		yamlPath := filepath.Join(skillDir, "skill.yaml")
		if testCase.yaml == "" {
			_ = os.Remove(yamlPath)
		} else if err := os.WriteFile(yamlPath, []byte(testCase.yaml), 0o644); err != nil {
			t.Fatalf("%s: write skill.yaml: %v", testCase.name, err)
		}
		_, err := Publish(workDir, skillDir, Options{Registry: "company"})
		if err == nil || !strings.Contains(err.Error(), testCase.want) {
			t.Errorf("%s: error = %v, want substring %q", testCase.name, err, testCase.want)
		}
	}
}

func TestPublishDestUnderSkillsTree(t *testing.T) {
	workDir, registryDir := setupLocalRegistry(t)
	if err := os.MkdirAll(filepath.Join(registryDir, "skills"), 0o755); err != nil {
		t.Fatalf("mkdir skills: %v", err)
	}
	skillDir := writeSkill(t, t.TempDir(), "review", "1.0.0", "# review\n")
	result, err := Publish(workDir, skillDir, Options{Registry: "company"})
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if result.Dest != "skills/acme/review" {
		t.Errorf("Dest = %q, want skills/acme/review", result.Dest)
	}
	if _, err := os.Stat(filepath.Join(registryDir, "skills", "acme", "review", "SKILL.md")); err != nil {
		t.Errorf("published files missing: %v", err)
	}
}

func TestPublishRejectsEscapingDest(t *testing.T) {
	workDir, _ := setupLocalRegistry(t)
	skillDir := writeSkill(t, t.TempDir(), "review", "1.0.0", "# review\n")
	if _, err := Publish(workDir, skillDir, Options{Registry: "company", Dest: "../outside"}); err == nil {
		t.Error("expected error for escaping dest")
	}
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}

func setupGitRegistry(t *testing.T) (string, string) {
	t.Helper()
	gitConfig := filepath.Join(t.TempDir(), "gitconfig")
	if err := os.WriteFile(gitConfig, []byte("[user]\n\tname = Test\n\temail = test@example.com\n"), 0o644); err != nil {
		t.Fatalf("write gitconfig: %v", err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", gitConfig)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	t.Setenv("SKILLHUB_HOME", t.TempDir())

	bare := filepath.Join(t.TempDir(), "registry.git")
	if err := os.MkdirAll(bare, 0o755); err != nil {
		t.Fatalf("mkdir bare: %v", err)
	}
	gitRun(t, bare, "init", "--bare", "--initial-branch=main")
	seed := filepath.Join(t.TempDir(), "seed")
	gitRun(t, filepath.Dir(seed), "clone", bare, seed)
	if err := os.WriteFile(filepath.Join(seed, "README.md"), []byte("# registry\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	gitRun(t, seed, "add", "-A")
	gitRun(t, seed, "commit", "-m", "init")
	gitRun(t, seed, "push", "origin", "HEAD")

	workDir := t.TempDir()
	cfg, err := config.Load(workDir)
	if err != nil {
		t.Fatalf("config load: %v", err)
	}
	cfg.Registries["team"] = config.Registry{Type: "git", URL: bare}
	if err := config.Save(workDir, cfg); err != nil {
		t.Fatalf("config save: %v", err)
	}
	return workDir, bare
}

func TestPublishGitPushesDefaultBranch(t *testing.T) {
	workDir, bare := setupGitRegistry(t)
	skillDir := writeSkill(t, t.TempDir(), "review", "1.0.0", "# review\n")
	result, err := Publish(workDir, skillDir, Options{Registry: "team"})
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if !result.Pushed {
		t.Error("expected push")
	}

	verify := filepath.Join(t.TempDir(), "verify")
	gitRun(t, filepath.Dir(verify), "clone", bare, verify)
	index, err := registry.LoadIndex(verify)
	if err != nil {
		t.Fatalf("LoadIndex from pushed registry: %v", err)
	}
	if len(index.Skills) != 1 || index.Skills[0].Identity != "acme/review" {
		t.Fatalf("pushed index skills = %+v", index.Skills)
	}
	if _, err := os.Stat(filepath.Join(verify, "review", "SKILL.md")); err != nil {
		t.Errorf("pushed files missing: %v", err)
	}
	if _, err := registry.ValidateLoadedIndex(verify, "team"); err != nil {
		t.Errorf("pushed registry failed validation: %v", err)
	}
}

func TestPublishGitPushesReviewBranch(t *testing.T) {
	workDir, bare := setupGitRegistry(t)
	skillDir := writeSkill(t, t.TempDir(), "review", "1.0.0", "# review\n")
	result, err := Publish(workDir, skillDir, Options{Registry: "team", Branch: "publish/review"})
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if !result.Pushed || result.Branch != "publish/review" {
		t.Errorf("unexpected result: %+v", result)
	}

	verify := filepath.Join(t.TempDir(), "verify")
	gitRun(t, filepath.Dir(verify), "clone", "--branch", "publish/review", bare, verify)
	if _, err := registry.LoadIndex(verify); err != nil {
		t.Fatalf("LoadIndex from review branch: %v", err)
	}
	mainVerify := filepath.Join(t.TempDir(), "main-verify")
	gitRun(t, filepath.Dir(mainVerify), "clone", "--branch", "main", bare, mainVerify)
	if _, err := os.Stat(filepath.Join(mainVerify, registry.IndexFileName)); !os.IsNotExist(err) {
		t.Errorf("main branch should not have the index yet: %v", err)
	}
}

func TestParseGitHubRepo(t *testing.T) {
	cases := []struct {
		url   string
		owner string
		repo  string
		ok    bool
	}{
		{"https://github.com/acme/skills.git", "acme", "skills", true},
		{"https://github.com/acme/skills", "acme", "skills", true},
		{"git@github.com:acme/skills.git", "acme", "skills", true},
		{"ssh://git@github.com/acme/skills.git", "acme", "skills", true},
		{"https://gitlab.com/acme/skills.git", "", "", false},
		{"https://github.com/acme", "", "", false},
	}
	for _, testCase := range cases {
		owner, repo, err := parseGitHubRepo(testCase.url)
		if testCase.ok && (err != nil || owner != testCase.owner || repo != testCase.repo) {
			t.Errorf("parseGitHubRepo(%q) = (%q, %q, %v)", testCase.url, owner, repo, err)
		}
		if !testCase.ok && err == nil {
			t.Errorf("parseGitHubRepo(%q) expected error", testCase.url)
		}
	}
}

type ghCall struct {
	args []string
}

func stubGH(t *testing.T, login string, prURL string) *[]ghCall {
	t.Helper()
	calls := &[]ghCall{}
	previousRunGH := runGH
	previousResolve := resolveGitHubRepo
	runGH = func(dir string, args ...string) (string, error) {
		*calls = append(*calls, ghCall{args: args})
		switch args[0] {
		case "api":
			return login + "\n", nil
		case "repo":
			return "created fork\n", nil
		case "pr":
			return prURL + "\n", nil
		default:
			return "", nil
		}
	}
	resolveGitHubRepo = func(url string) (string, string, error) {
		return "acme", "registry", nil
	}
	t.Cleanup(func() {
		runGH = previousRunGH
		resolveGitHubRepo = previousResolve
	})
	return calls
}

func lastGHArgs(calls []ghCall) []string {
	if len(calls) == 0 {
		return nil
	}
	return calls[len(calls)-1].args
}

func TestPublishPRForkFlow(t *testing.T) {
	workDir, _ := setupGitRegistry(t)
	forkBare := filepath.Join(t.TempDir(), "fork.git")
	if err := os.MkdirAll(forkBare, 0o755); err != nil {
		t.Fatalf("mkdir fork bare: %v", err)
	}
	gitRun(t, forkBare, "init", "--bare", "--initial-branch=main")
	calls := stubGH(t, "contributor", "https://github.com/acme/registry/pull/7")
	previousForkURL := forkPushURL
	forkPushURL = func(login string, repo string) string { return forkBare }
	t.Cleanup(func() { forkPushURL = previousForkURL })

	skillDir := writeSkill(t, t.TempDir(), "review", "1.0.0", "# review\n")
	result, err := Publish(workDir, skillDir, Options{Registry: "team", PR: true})
	if err != nil {
		t.Fatalf("Publish --pr: %v", err)
	}
	if result.PRURL != "https://github.com/acme/registry/pull/7" {
		t.Errorf("PRURL = %q", result.PRURL)
	}
	wantBranch := "publish/acme__review-1.0.0"
	if result.Branch != wantBranch {
		t.Errorf("Branch = %q, want %q", result.Branch, wantBranch)
	}

	prArgs := lastGHArgs(*calls)
	if len(prArgs) < 6 || prArgs[0] != "pr" || prArgs[1] != "create" {
		t.Fatalf("last gh call = %v", prArgs)
	}
	joined := strings.Join(prArgs, " ")
	if !strings.Contains(joined, "--repo acme/registry") || !strings.Contains(joined, "--head contributor:"+wantBranch) {
		t.Errorf("gh pr create args = %v", prArgs)
	}

	verify := filepath.Join(t.TempDir(), "verify-fork")
	gitRun(t, filepath.Dir(verify), "clone", "--branch", wantBranch, forkBare, verify)
	if _, err := registry.LoadIndex(verify); err != nil {
		t.Fatalf("LoadIndex from fork branch: %v", err)
	}
}

func TestPublishPROwnerPushesUpstream(t *testing.T) {
	workDir, bare := setupGitRegistry(t)
	calls := stubGH(t, "acme", "https://github.com/acme/registry/pull/8")

	skillDir := writeSkill(t, t.TempDir(), "review", "1.0.0", "# review\n")
	result, err := Publish(workDir, skillDir, Options{Registry: "team", PR: true, Branch: "publish/review"})
	if err != nil {
		t.Fatalf("Publish --pr as owner: %v", err)
	}
	if result.PRURL == "" || result.Branch != "publish/review" {
		t.Errorf("unexpected result: %+v", result)
	}
	joined := strings.Join(lastGHArgs(*calls), " ")
	if !strings.Contains(joined, "--head publish/review") || strings.Contains(joined, "--head acme:") {
		t.Errorf("owner flow should push upstream head: %v", joined)
	}

	verify := filepath.Join(t.TempDir(), "verify-upstream")
	gitRun(t, filepath.Dir(verify), "clone", "--branch", "publish/review", bare, verify)
	if _, err := registry.LoadIndex(verify); err != nil {
		t.Fatalf("LoadIndex from upstream branch: %v", err)
	}
}

func TestPublishPRRequiresGitRegistry(t *testing.T) {
	workDir, _ := setupLocalRegistry(t)
	skillDir := writeSkill(t, t.TempDir(), "review", "1.0.0", "# review\n")
	if _, err := Publish(workDir, skillDir, Options{Registry: "company", PR: true}); err == nil || !strings.Contains(err.Error(), "--pr requires a git registry") {
		t.Errorf("expected git registry error, got %v", err)
	}
}

func writeSkillWithRequires(t *testing.T, dir string, requires string) string {
	t.Helper()
	skillDir := filepath.Join(dir, "review")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir skill: %v", err)
	}
	yaml := "name: review\nnamespace: acme\nversion: 1.0.0\ndescription: d\ntargets:\n- codex\nrequires:\n  skillhub: \"" + requires + "\"\n"
	if err := os.WriteFile(filepath.Join(skillDir, "skill.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatalf("write skill.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# review\n"), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	return skillDir
}

func TestPublishCarriesRequiresIntoIndex(t *testing.T) {
	workDir, registryDir := setupLocalRegistry(t)
	skillDir := writeSkillWithRequires(t, t.TempDir(), ">=1.4.0")
	if _, err := Publish(workDir, skillDir, Options{Registry: "company"}); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	index, err := registry.LoadIndex(registryDir)
	if err != nil {
		t.Fatalf("LoadIndex: %v", err)
	}
	if index.Skills[0].Requires["skillhub"] != ">=1.4.0" {
		t.Errorf("index requires = %v", index.Skills[0].Requires)
	}
}

func TestPublishRejectsInvalidRequires(t *testing.T) {
	workDir, _ := setupLocalRegistry(t)
	skillDir := writeSkillWithRequires(t, t.TempDir(), "~1.4.0")
	if _, err := Publish(workDir, skillDir, Options{Registry: "company"}); err == nil || !strings.Contains(err.Error(), "requires.skillhub") {
		t.Errorf("expected invalid constraint error, got %v", err)
	}
}
