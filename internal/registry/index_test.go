package registry

import (
	"path/filepath"
	"testing"
)

func TestRegistrySourcePathValid(t *testing.T) {
	root := t.TempDir()
	got, err := RegistrySourcePath(root, "skills/git-commit")
	if err != nil {
		t.Fatalf("RegistrySourcePath: %v", err)
	}
	want := filepath.Join(root, "skills", "git-commit")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRegistrySourcePathRejectsUnsafe(t *testing.T) {
	root := t.TempDir()
	// filepath.Abs yields a path that is absolute on the current OS, so the
	// absolute-path rejection is exercised correctly on Windows too.
	absPath, err := filepath.Abs("abs-thing")
	if err != nil {
		t.Fatal(err)
	}
	cases := []string{
		absPath,
		"..",
		"../escape",
		"skills/../../escape",
		"skills\\windows",
		".",
	}
	for _, in := range cases {
		if _, err := RegistrySourcePath(root, in); err == nil {
			t.Errorf("RegistrySourcePath(%q) = nil error, want rejection", in)
		}
	}
}

func TestValidTarget(t *testing.T) {
	for _, ok := range []string{TargetCodex, TargetClaude, TargetGemini} {
		if !validTarget(ok) {
			t.Errorf("validTarget(%q) = false, want true", ok)
		}
	}
	for _, bad := range []string{"hermes", "", "Codex"} {
		if validTarget(bad) {
			t.Errorf("validTarget(%q) = true, want false", bad)
		}
	}
}

func TestValidSourceType(t *testing.T) {
	for _, ok := range []string{SourceTypeRegistry, SourceTypeGit} {
		if !validSourceType(ok) {
			t.Errorf("validSourceType(%q) = false, want true", ok)
		}
	}
	if validSourceType("http") {
		t.Error("validSourceType(http) = true, want false")
	}
}

func TestValidTrustLevel(t *testing.T) {
	for _, ok := range []string{TrustOfficial, TrustCurated, TrustCommunity, TrustPrivate, TrustUnknown} {
		if !validTrustLevel(ok) {
			t.Errorf("validTrustLevel(%q) = false, want true", ok)
		}
	}
	if validTrustLevel("verified") {
		t.Error("validTrustLevel(verified) = true, want false")
	}
}

func TestMatches(t *testing.T) {
	skill := IndexSkill{
		Identity:    "official/git-commit",
		Name:        "git-commit",
		Description: "Generate commit messages",
		Tags:        []string{"git", "vcs"},
	}
	for _, q := range []string{"git-commit", "commit messages", "vcs", "official"} {
		if !matches(skill, q) {
			t.Errorf("matches(%q) = false, want true", q)
		}
	}
	if matches(skill, "kubernetes") {
		t.Error("matches(kubernetes) = true, want false")
	}
}

func TestMatchScoreOrdering(t *testing.T) {
	skill := IndexSkill{
		Identity:    "official/git-commit",
		Name:        "git-commit",
		Description: "helps with commit messages",
		Tags:        []string{"git", "vcs"},
	}
	exact := matchScore(skill, "git-commit")
	prefix := matchScore(skill, "git-com")
	// "vcs" is a tag but not a name/identity prefix, isolating the tag-exact tier.
	tagExact := matchScore(skill, "vcs")
	descr := matchScore(skill, "commit messages")
	none := matchScore(skill, "zzz")

	if !(exact < prefix && prefix < tagExact && tagExact < descr && descr < none) {
		t.Errorf("score ordering wrong: exact=%d prefix=%d tag=%d descr=%d none=%d",
			exact, prefix, tagExact, descr, none)
	}
}
