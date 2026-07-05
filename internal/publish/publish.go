package publish

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/CassianFlorin/skill-hub/internal/config"
	"github.com/CassianFlorin/skill-hub/internal/registry"
	"github.com/CassianFlorin/skill-hub/internal/skill"
)

const (
	ActionCreated   = "created"
	ActionUpdated   = "updated"
	ActionUnchanged = "unchanged"
)

type Options struct {
	Registry string
	Dest     string
	Trust    string
	Branch   string
	Message  string
	DryRun   bool
	PR       bool
}

type Result struct {
	Identity  string
	Version   string
	Action    string
	Registry  string
	Dest      string
	IndexPath string
	Checksum  string
	DryRun    bool
	OldEntry  *registry.IndexSkill
	NewEntry  registry.IndexSkill
	Pushed    bool
	Branch    string
	CommitMsg string
	PRURL     string
}

func Publish(workDir string, skillPath string, opts Options) (Result, error) {
	if opts.Registry == "" {
		return Result{}, fmt.Errorf("publish requires --registry")
	}
	sourcePath, err := absoluteFrom(workDir, skillPath)
	if err != nil {
		return Result{}, err
	}
	meta, err := skill.LoadMetadata(sourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			return Result{}, fmt.Errorf("publish requires skill.yaml in %s", sourcePath)
		}
		return Result{}, err
	}
	if err := validatePublishable(meta); err != nil {
		return Result{}, err
	}
	meta.Namespace = skill.ResolveNamespace(meta, "")
	identity := skill.Identity(meta.Namespace, meta.Name)
	checksum, err := skill.ChecksumDir(sourcePath)
	if err != nil {
		return Result{}, err
	}

	cfg, err := config.Load(workDir)
	if err != nil {
		return Result{}, err
	}
	reg, ok := cfg.Registries[opts.Registry]
	if !ok {
		return Result{}, fmt.Errorf("unknown registry %q", opts.Registry)
	}
	if opts.PR && reg.Type != "git" {
		return Result{}, fmt.Errorf("--pr requires a git registry, %q is %s", opts.Registry, reg.Type)
	}

	var root string
	var cleanup func()
	switch reg.Type {
	case "local":
		root = reg.Path
		if info, err := os.Stat(root); err != nil || !info.IsDir() {
			return Result{}, fmt.Errorf("local registry path %s is not a directory", root)
		}
	case "git":
		tempRoot, tempCleanup, err := cloneForPublish(reg.URL)
		if err != nil {
			return Result{}, err
		}
		root = tempRoot
		cleanup = tempCleanup
	default:
		return Result{}, fmt.Errorf("unsupported registry type %q", reg.Type)
	}
	if cleanup != nil {
		defer cleanup()
	}

	index, err := loadOrInitIndex(root, opts.Registry)
	if err != nil {
		return Result{}, err
	}
	oldEntry, oldIndexPos := findEntry(index, identity)
	if oldEntry != nil && oldEntry.Source.Type != registry.SourceTypeRegistry {
		return Result{}, fmt.Errorf("%s already indexed with source type %q; publish only manages registry-source entries", identity, oldEntry.Source.Type)
	}
	if oldEntry != nil && oldEntry.Version == meta.Version {
		if oldEntry.Checksum == checksum {
			return Result{
				Identity: identity,
				Version:  meta.Version,
				Action:   ActionUnchanged,
				Registry: opts.Registry,
				Dest:     oldEntry.Source.Path,
				Checksum: checksum,
				DryRun:   opts.DryRun,
				OldEntry: oldEntry,
				NewEntry: *oldEntry,
			}, nil
		}
		return Result{}, fmt.Errorf("%s@%s already published with different content; bump the version in skill.yaml", identity, meta.Version)
	}
	if oldEntry != nil {
		if cmp, comparable := compareSemver(meta.Version, oldEntry.Version); comparable && cmp < 0 {
			return Result{}, fmt.Errorf("%s version %s is lower than published %s; publish a higher version", identity, meta.Version, oldEntry.Version)
		}
	}

	dest, err := resolveDest(root, opts.Dest, oldEntry, meta)
	if err != nil {
		return Result{}, err
	}
	newEntry := buildEntry(meta, identity, checksum, dest, oldEntry, opts.Trust)
	if err := validateTrust(newEntry.Trust.Level); err != nil {
		return Result{}, err
	}

	action := ActionCreated
	if oldEntry != nil {
		action = ActionUpdated
	}
	result := Result{
		Identity: identity,
		Version:  meta.Version,
		Action:   action,
		Registry: opts.Registry,
		Dest:     dest,
		Checksum: checksum,
		DryRun:   opts.DryRun,
		OldEntry: oldEntry,
		NewEntry: newEntry,
		Branch:   opts.Branch,
	}
	if opts.DryRun {
		return result, nil
	}

	destPath, err := registry.RegistrySourcePath(root, dest)
	if err != nil {
		return Result{}, err
	}
	if err := copyDir(sourcePath, destPath); err != nil {
		return Result{}, err
	}
	if oldEntry != nil && oldEntry.Source.Path != dest {
		oldPath, err := registry.RegistrySourcePath(root, oldEntry.Source.Path)
		if err == nil {
			_ = os.RemoveAll(oldPath)
		}
	}

	if oldIndexPos >= 0 {
		index.Skills[oldIndexPos] = newEntry
	} else {
		index.Skills = append(index.Skills, newEntry)
		sort.Slice(index.Skills, func(i, j int) bool {
			return index.Skills[i].Identity < index.Skills[j].Identity
		})
	}
	index.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	if err := registry.WriteIndex(root, index); err != nil {
		return Result{}, err
	}
	if _, err := registry.ValidateLoadedIndex(root, opts.Registry); err != nil {
		return Result{}, fmt.Errorf("published index failed validation: %w", err)
	}
	result.IndexPath = filepath.Join(root, registry.IndexFileName)
	if reg.Type == "local" {
		return result, nil
	}

	message := opts.Message
	if message == "" {
		message = fmt.Sprintf("publish %s@%s", identity, meta.Version)
	}
	result.CommitMsg = message
	branch := opts.Branch
	if opts.PR && branch == "" {
		branch = "publish/" + skill.SafeIdentity(identity) + "-" + meta.Version
	}
	result.Branch = branch
	if branch != "" {
		if err := runGit(root, "checkout", "-B", branch); err != nil {
			return Result{}, err
		}
	}
	if err := runGit(root, "add", "-A"); err != nil {
		return Result{}, err
	}
	if err := runGit(root, "commit", "-m", message); err != nil {
		return Result{}, err
	}
	if opts.PR {
		prURL, err := createPullRequest(root, prRequest{
			RegistryURL: reg.URL,
			Branch:      branch,
			Title:       message,
			Identity:    identity,
			Version:     meta.Version,
			Dest:        dest,
			Checksum:    checksum,
		})
		if err != nil {
			return Result{}, err
		}
		result.Pushed = true
		result.PRURL = prURL
		return result, nil
	}
	if err := runGit(root, "push", "origin", "HEAD"); err != nil {
		return Result{}, fmt.Errorf("%w\nif you lack write access to this registry, use --pr to publish through a fork and pull request, or --branch to push a review branch when you have write access", err)
	}
	result.Pushed = true
	return result, nil
}

func validatePublishable(meta skill.Metadata) error {
	if meta.Version == "" || meta.Version == skill.Unversioned {
		return fmt.Errorf("skill.yaml must set an explicit version before publish")
	}
	if meta.Description == "" {
		return fmt.Errorf("skill.yaml must set description before publish")
	}
	if len(meta.Targets) == 0 {
		return fmt.Errorf("skill.yaml must set at least one target before publish")
	}
	for _, target := range meta.Targets {
		if !registry.ValidTarget(target) {
			return fmt.Errorf("skill.yaml target %q is not supported", target)
		}
	}
	if meta.Namespace == "" && meta.Author == "" {
		return fmt.Errorf("skill.yaml must set namespace or author before publish")
	}
	meta.Namespace = skill.ResolveNamespace(meta, "")
	if meta.Namespace == "" {
		return fmt.Errorf("could not resolve a namespace for this skill")
	}
	return nil
}

// compareSemver returns -1/0/1 when both versions parse as semver
// (optional "v" prefix, MAJOR.MINOR.PATCH, optional -prerelease).
// comparable is false when either version does not parse, in which
// case the caller should skip ordering checks.
func compareSemver(left string, right string) (int, bool) {
	leftParts, leftPre, ok := parseSemver(left)
	if !ok {
		return 0, false
	}
	rightParts, rightPre, ok := parseSemver(right)
	if !ok {
		return 0, false
	}
	for i := 0; i < 3; i++ {
		if leftParts[i] != rightParts[i] {
			if leftParts[i] < rightParts[i] {
				return -1, true
			}
			return 1, true
		}
	}
	switch {
	case leftPre == rightPre:
		return 0, true
	case leftPre == "":
		return 1, true
	case rightPre == "":
		return -1, true
	case leftPre < rightPre:
		return -1, true
	default:
		return 1, true
	}
}

func parseSemver(version string) ([3]int, string, bool) {
	version = strings.TrimPrefix(version, "v")
	version, prerelease, _ := strings.Cut(version, "-")
	version, _, _ = strings.Cut(version, "+")
	fields := strings.Split(version, ".")
	if len(fields) != 3 {
		return [3]int{}, "", false
	}
	var parts [3]int
	for i, field := range fields {
		if field == "" {
			return [3]int{}, "", false
		}
		value := 0
		for _, char := range field {
			if char < '0' || char > '9' {
				return [3]int{}, "", false
			}
			value = value*10 + int(char-'0')
		}
		parts[i] = value
	}
	return parts, prerelease, true
}

func validateTrust(level string) error {
	if registry.ValidTrustLevel(level) {
		return nil
	}
	return fmt.Errorf("unsupported trust level %q", level)
}

func loadOrInitIndex(root string, name string) (registry.Index, error) {
	index, err := registry.LoadIndex(root)
	if err != nil {
		if os.IsNotExist(err) {
			return registry.Index{
				SchemaVersion: registry.IndexSchemaVersion,
				Registry:      name,
			}, nil
		}
		return registry.Index{}, err
	}
	return index, nil
}

func findEntry(index registry.Index, identity string) (*registry.IndexSkill, int) {
	for i := range index.Skills {
		if index.Skills[i].Identity == identity {
			entry := index.Skills[i]
			return &entry, i
		}
	}
	return nil, -1
}

func resolveDest(root string, dest string, oldEntry *registry.IndexSkill, meta skill.Metadata) (string, error) {
	if dest == "" {
		switch {
		case oldEntry != nil:
			dest = oldEntry.Source.Path
		default:
			if info, err := os.Stat(filepath.Join(root, "skills")); err == nil && info.IsDir() {
				dest = "skills/" + meta.Namespace + "/" + meta.Name
			} else {
				dest = meta.Name
			}
		}
	}
	dest = filepath.ToSlash(filepath.Clean(dest))
	if _, err := registry.RegistrySourcePath(root, dest); err != nil {
		return "", err
	}
	if dest == registry.IndexFileName {
		return "", fmt.Errorf("destination %q conflicts with the registry index", dest)
	}
	return dest, nil
}

func buildEntry(meta skill.Metadata, identity string, checksum string, dest string, oldEntry *registry.IndexSkill, trust string) registry.IndexSkill {
	now := time.Now().UTC().Format(time.RFC3339)
	entry := registry.IndexSkill{
		Identity:    identity,
		Name:        meta.Name,
		Namespace:   meta.Namespace,
		Version:     meta.Version,
		Description: meta.Description,
		Targets:     meta.Targets,
		Tags:        meta.Tags,
		Source:      registry.IndexSource{Type: registry.SourceTypeRegistry, Path: dest},
		Trust:       registry.IndexTrust{Level: registry.TrustPrivate},
		UpdatedAt:   now,
		Checksum:    checksum,
	}
	if meta.Author != "" {
		entry.Maintainers = []string{meta.Author}
	}
	if oldEntry != nil {
		entry.Trust = oldEntry.Trust
		entry.Featured = oldEntry.Featured
		entry.License = oldEntry.License
		if len(entry.Maintainers) == 0 {
			entry.Maintainers = oldEntry.Maintainers
		}
	}
	if trust != "" {
		entry.Trust = registry.IndexTrust{Level: trust}
	}
	return entry
}

func cloneForPublish(url string) (string, func(), error) {
	tempDir, err := os.MkdirTemp("", "skillhub-publish-")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(tempDir) }
	if err := runGit("", "clone", url, tempDir); err != nil {
		cleanup()
		return "", nil, err
	}
	return tempDir, cleanup, nil
}

func runGit(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %v failed: %w\n%s", args, err, strings.TrimSpace(string(output)))
	}
	return nil
}

func absoluteFrom(workDir string, path string) (string, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(workDir, path)
	}
	return filepath.Abs(path)
}

func copyDir(src string, dst string) error {
	if err := os.RemoveAll(dst); err != nil {
		return err
	}
	return filepath.WalkDir(src, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
}
