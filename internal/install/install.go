package install

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cassian/skill-hub/internal/config"
	"github.com/cassian/skill-hub/internal/registry"
	"github.com/cassian/skill-hub/internal/skill"
)

const LockFileName = "skillhub.lock"

type LockFile struct {
	Skills []LockedSkill `json:"skills"`
}

type LockedSkill struct {
	Identity       string     `json:"identity"`
	Name           string     `json:"name"`
	Namespace      string     `json:"namespace"`
	Version        string     `json:"version"`
	Description    string     `json:"description,omitempty"`
	SourceType     string     `json:"source_type"`
	SourceRegistry string     `json:"source_registry,omitempty"`
	SourceURL      string     `json:"source_url,omitempty"`
	SourceRef      string     `json:"source_ref,omitempty"`
	SourceCommit   string     `json:"source_commit,omitempty"`
	SourcePath     string     `json:"source_path"`
	SourceSubpath  string     `json:"source_subpath,omitempty"`
	SourceCache    string     `json:"source_cache,omitempty"`
	Checksum       string     `json:"checksum,omitempty"`
	InstalledPath  string     `json:"installed_path"`
	Targets        []string   `json:"targets,omitempty"`
	DeployedTo     []string   `json:"deployed_runtimes,omitempty"`
	Hold           *HoldState `json:"hold,omitempty"`
	UpdatedAt      string     `json:"updated_at"`
}

type HoldState struct {
	Reason    string `json:"reason,omitempty"`
	CreatedAt string `json:"created_at"`
}

type UpdateOptions struct {
	Identity string
}

type UpdatePlan struct {
	Identity         string
	CurrentVersion   string
	AvailableVersion string
	CurrentCommit    string
	AvailableCommit  string
	SourceType       string
	SourceRegistry   string
	SourceURL        string
	AvailablePath    string
	Targets          []string
	DeployedTo       []string
	Held             bool
	HoldReason       string
}

type RollbackOptions struct {
	To string
}

type HistoryEntry struct {
	Identity      string
	Version       string
	SourceRef     string
	SourceCommit  string
	CreatedAt     string
	InstalledPath string
}

func LockPath() (string, error) {
	home, err := config.DefaultHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, LockFileName), nil
}

func LoadLock() (LockFile, error) {
	path, err := LockPath()
	if err != nil {
		return LockFile{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return LockFile{}, nil
		}
		return LockFile{}, err
	}
	var lock LockFile
	if err := json.Unmarshal(data, &lock); err != nil {
		return LockFile{}, err
	}
	return lock, nil
}

func SaveLock(lock LockFile) error {
	path, err := LockPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func Install(workDir string, spec string) (LockedSkill, error) {
	cfg, err := config.Load(workDir)
	if err != nil {
		return LockedSkill{}, err
	}
	source, err := resolveSource(workDir, cfg, spec)
	if err != nil {
		return LockedSkill{}, err
	}
	meta, err := skill.LoadCompatibleMetadata(source.Path, source.Registry)
	if err != nil {
		return LockedSkill{}, err
	}
	identity := skill.Identity(meta.Namespace, meta.Name)
	if source.Ref != "" && source.Type != registry.SourceTypeGit && meta.Version != source.Ref {
		return LockedSkill{}, fmt.Errorf("version %s not available for %s", source.Ref, identity)
	}
	lock, err := LoadLock()
	if err != nil {
		return LockedSkill{}, err
	}
	if existing, ok := lock.find(identity); ok {
		if err := saveHistory(existing); err != nil {
			return LockedSkill{}, err
		}
	}
	installedPath := filepath.Join(cfg.InstallDir, skill.SafeIdentity(identity))
	if err := copyDir(source.Path, installedPath); err != nil {
		return LockedSkill{}, err
	}
	if meta.Generated {
		if err := skill.WriteGeneratedMetadata(installedPath, meta); err != nil {
			return LockedSkill{}, err
		}
	}
	checksum, err := skill.ChecksumDir(installedPath)
	if err != nil {
		return LockedSkill{}, err
	}
	locked := LockedSkill{
		Identity:       identity,
		Name:           meta.Name,
		Namespace:      meta.Namespace,
		Version:        meta.Version,
		Description:    meta.Description,
		SourceType:     source.Type,
		SourceRegistry: source.Registry,
		SourceURL:      source.URL,
		SourceRef:      source.Ref,
		SourceCommit:   source.Commit,
		SourcePath:     source.Path,
		SourceSubpath:  source.Subpath,
		SourceCache:    source.CacheName,
		Checksum:       checksum,
		InstalledPath:  installedPath,
		Targets:        meta.Targets,
		UpdatedAt:      time.Now().UTC().Format(time.RFC3339),
	}
	lock.upsert(locked)
	if err := SaveLock(lock); err != nil {
		return LockedSkill{}, err
	}
	return locked, nil
}

func Rollback(identity string) (LockedSkill, error) {
	return RollbackWithOptions(identity, RollbackOptions{})
}

func RollbackWithOptions(identity string, options RollbackOptions) (LockedSkill, error) {
	lock, err := LoadLock()
	if err != nil {
		return LockedSkill{}, err
	}
	current, ok := lock.find(identity)
	if !ok {
		return LockedSkill{}, fmt.Errorf("unknown installed skill %q", identity)
	}
	snapshot, err := historySnapshotForVersion(current.DisplayIdentity(), options.To)
	if err != nil {
		return LockedSkill{}, err
	}
	if err := copyDir(snapshot.InstalledPath, current.InstalledPath); err != nil {
		return LockedSkill{}, err
	}
	restored := snapshot.Locked
	restored.InstalledPath = current.InstalledPath
	restored.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	lock.upsert(restored)
	if err := SaveLock(lock); err != nil {
		return LockedSkill{}, err
	}
	return restored, nil
}

func History(identity string) ([]HistoryEntry, error) {
	lock, err := LoadLock()
	if err != nil {
		return nil, err
	}
	current, ok := lock.find(identity)
	if !ok {
		return nil, fmt.Errorf("unknown installed skill %q", identity)
	}
	snapshots, err := historySnapshots(current.DisplayIdentity())
	if err != nil {
		return nil, err
	}
	entries := make([]HistoryEntry, 0, len(snapshots))
	for _, snapshot := range snapshots {
		entries = append(entries, HistoryEntry{
			Identity:      snapshot.Locked.DisplayIdentity(),
			Version:       snapshot.Locked.Version,
			SourceRef:     snapshot.Locked.SourceRef,
			SourceCommit:  snapshot.Locked.SourceCommit,
			CreatedAt:     snapshot.CreatedAt,
			InstalledPath: snapshot.InstalledPath,
		})
	}
	return entries, nil
}

func Uninstall(identity string) (LockedSkill, error) {
	lock, err := LoadLock()
	if err != nil {
		return LockedSkill{}, err
	}
	locked, ok := lock.find(identity)
	if !ok {
		return LockedSkill{}, fmt.Errorf("unknown installed skill %q", identity)
	}
	if locked.InstalledPath != "" {
		if err := os.RemoveAll(locked.InstalledPath); err != nil {
			return LockedSkill{}, err
		}
	}
	lock.remove(locked.DisplayIdentity())
	if err := SaveLock(lock); err != nil {
		return LockedSkill{}, err
	}
	return locked, nil
}

func (l *LockFile) find(identity string) (LockedSkill, bool) {
	for _, existing := range l.Skills {
		if existing.DisplayIdentity() == identity || existing.Name == identity {
			return existing, true
		}
	}
	return LockedSkill{}, false
}

func (l *LockFile) remove(identity string) {
	var kept []LockedSkill
	for _, existing := range l.Skills {
		if existing.DisplayIdentity() != identity {
			kept = append(kept, existing)
		}
	}
	l.Skills = kept
}

func Hold(identity string, reason string) (LockedSkill, error) {
	lock, err := LoadLock()
	if err != nil {
		return LockedSkill{}, err
	}
	for i, locked := range lock.Skills {
		if locked.DisplayIdentity() != identity && locked.Name != identity {
			continue
		}
		lock.Skills[i].Hold = &HoldState{Reason: reason, CreatedAt: time.Now().UTC().Format(time.RFC3339)}
		if err := SaveLock(lock); err != nil {
			return LockedSkill{}, err
		}
		return lock.Skills[i], nil
	}
	return LockedSkill{}, fmt.Errorf("unknown installed skill %q", identity)
}

func Unhold(identity string) (LockedSkill, error) {
	lock, err := LoadLock()
	if err != nil {
		return LockedSkill{}, err
	}
	for i, locked := range lock.Skills {
		if locked.DisplayIdentity() != identity && locked.Name != identity {
			continue
		}
		lock.Skills[i].Hold = nil
		if err := SaveLock(lock); err != nil {
			return LockedSkill{}, err
		}
		return lock.Skills[i], nil
	}
	return LockedSkill{}, fmt.Errorf("unknown installed skill %q", identity)
}

func HeldSkills() ([]LockedSkill, error) {
	lock, err := LoadLock()
	if err != nil {
		return nil, err
	}
	var held []LockedSkill
	for _, locked := range lock.Skills {
		if locked.Hold != nil {
			held = append(held, locked)
		}
	}
	return held, nil
}

func UpdateAll() ([][3]string, error) {
	lock, err := LoadLock()
	if err != nil {
		return nil, err
	}
	changes, err := updateLock(&lock, false)
	if err != nil {
		return nil, err
	}
	if err := SaveLock(lock); err != nil {
		return nil, err
	}
	return changes, nil
}

func UpdateOne(identity string) ([][3]string, error) {
	lock, err := LoadLock()
	if err != nil {
		return nil, err
	}
	if _, ok := lock.find(identity); !ok {
		return nil, fmt.Errorf("unknown installed skill %q", identity)
	}
	changes, err := updateLock(&lock, false, identity)
	if err != nil {
		return nil, err
	}
	if err := SaveLock(lock); err != nil {
		return nil, err
	}
	return changes, nil
}

func PlanUpdates() ([][3]string, error) {
	plans, err := PlanUpdateDetails(UpdateOptions{})
	if err != nil {
		return nil, err
	}
	changes := make([][3]string, 0, len(plans))
	for _, plan := range plans {
		changes = append(changes, [3]string{plan.Identity, plan.CurrentVersion, plan.AvailableVersion})
	}
	return changes, nil
}

func PlanUpdateDetails(options UpdateOptions) ([]UpdatePlan, error) {
	lock, err := LoadLock()
	if err != nil {
		return nil, err
	}
	return planUpdateDetails(lock, options)
}

func planUpdateDetails(lock LockFile, options UpdateOptions) ([]UpdatePlan, error) {
	var plans []UpdatePlan
	for _, locked := range lock.Skills {
		identity := locked.DisplayIdentity()
		if options.Identity != "" && identity != options.Identity && locked.Name != options.Identity {
			continue
		}
		resolved, err := resolveUpdateSource(locked, true)
		if err != nil {
			return nil, err
		}
		meta, err := skill.LoadCompatibleMetadata(resolved.SourcePath, resolved.SourceRegistry)
		if err != nil {
			return nil, fmt.Errorf("update %s: %w", locked.Name, err)
		}
		if meta.Version == locked.Version {
			continue
		}
		plan := UpdatePlan{
			Identity:         identity,
			CurrentVersion:   locked.Version,
			AvailableVersion: meta.Version,
			CurrentCommit:    locked.SourceCommit,
			AvailableCommit:  resolved.SourceCommit,
			SourceType:       locked.SourceType,
			SourceRegistry:   locked.SourceRegistry,
			SourceURL:        locked.SourceURL,
			AvailablePath:    resolved.SourcePath,
			Targets:          meta.Targets,
			DeployedTo:       locked.DeployedTo,
		}
		if locked.Hold != nil {
			plan.Held = true
			plan.HoldReason = locked.Hold.Reason
		}
		plans = append(plans, plan)
	}
	if options.Identity != "" && len(plans) == 0 {
		if _, ok := lock.find(options.Identity); !ok {
			return nil, fmt.Errorf("unknown installed skill %q", options.Identity)
		}
	}
	return plans, nil
}

func updateLock(lock *LockFile, dryRun bool, identities ...string) ([][3]string, error) {
	filter := ""
	if len(identities) > 0 {
		filter = identities[0]
	}
	var changes [][3]string
	for i, locked := range lock.Skills {
		if filter != "" && locked.DisplayIdentity() != filter && locked.Name != filter {
			continue
		}
		resolved, err := resolveUpdateSource(locked, dryRun)
		if err != nil {
			return nil, err
		}
		if !dryRun {
			lock.Skills[i].SourcePath = resolved.SourcePath
			lock.Skills[i].SourceCommit = resolved.SourceCommit
			lock.Skills[i].SourceSubpath = resolved.SourceSubpath
			lock.Skills[i].SourceCache = resolved.SourceCache
		}
		meta, err := skill.LoadCompatibleMetadata(resolved.SourcePath, resolved.SourceRegistry)
		if err != nil {
			return nil, fmt.Errorf("update %s: %w", locked.Name, err)
		}
		if meta.Version == locked.Version {
			continue
		}
		if locked.Hold != nil {
			continue
		}
		changes = append(changes, [3]string{locked.displayIdentity(), locked.Version, meta.Version})
		if dryRun {
			continue
		}
		if err := saveHistory(locked); err != nil {
			return nil, err
		}
		if err := copyDir(resolved.SourcePath, locked.InstalledPath); err != nil {
			return nil, err
		}
		if meta.Generated {
			if err := skill.WriteGeneratedMetadata(locked.InstalledPath, meta); err != nil {
				return nil, err
			}
		}
		checksum, err := skill.ChecksumDir(locked.InstalledPath)
		if err != nil {
			return nil, err
		}
		identity := skill.Identity(meta.Namespace, meta.Name)
		lock.Skills[i].Identity = identity
		lock.Skills[i].Name = meta.Name
		lock.Skills[i].Namespace = meta.Namespace
		lock.Skills[i].Version = meta.Version
		lock.Skills[i].Description = meta.Description
		lock.Skills[i].Targets = meta.Targets
		lock.Skills[i].Checksum = checksum
		lock.Skills[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	return changes, nil
}

func resolveUpdateSource(locked LockedSkill, dryRun bool) (LockedSkill, error) {
	resolved := locked
	if locked.SourceType != registry.SourceTypeGit {
		return resolved, nil
	}
	cacheName := locked.SourceCache
	sourceSubpath := locked.SourceSubpath
	if inferredCache, inferredSubpath, ok := inferCachedSource(locked.SourcePath, locked.SourceRef); ok {
		if cacheName == "" {
			cacheName = inferredCache
		}
		if sourceSubpath == "" {
			sourceSubpath = inferredSubpath
		}
	}
	if cacheName == "" {
		cacheName = locked.SourceRegistry
	}
	if sourceSubpath == "" {
		sourceSubpath = locked.Name
	}
	var cachePath string
	var commit string
	var err error
	if locked.SourceRef != "" {
		cachePath, commit, err = registry.EnsureGitCacheAtRef(cacheName, locked.SourceURL, locked.SourceRef)
	} else {
		cachePath, err = registry.EnsureGitCache(cacheName, locked.SourceURL)
		if err == nil {
			commit, err = registry.GitCommit(cachePath)
		}
	}
	if err != nil {
		return LockedSkill{}, fmt.Errorf("update %s: %w", locked.Name, err)
	}
	resolvedPath, err := registry.RegistrySourcePath(cachePath, sourceSubpath)
	if err != nil {
		return LockedSkill{}, fmt.Errorf("update %s: %w", locked.Name, err)
	}
	resolved.SourcePath = resolvedPath
	resolved.SourceCommit = commit
	resolved.SourceSubpath = filepath.ToSlash(sourceSubpath)
	resolved.SourceCache = cacheName
	return resolved, nil
}

func inferCachedSource(sourcePath string, sourceRef string) (cacheName string, sourceSubpath string, ok bool) {
	if sourcePath == "" {
		return "", "", false
	}
	dir := sourcePath
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			rel, err := filepath.Rel(dir, sourcePath)
			if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
				return "", "", false
			}
			name := filepath.Base(dir)
			if sourceRef != "" {
				suffix := "__" + skill.SafeIdentity(sourceRef)
				name = strings.TrimSuffix(name, suffix)
			}
			return name, filepath.ToSlash(rel), true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", "", false
		}
		dir = parent
	}
}

type historyManifest struct {
	Locked LockedSkill `json:"locked"`
}

type historySnapshot struct {
	Locked        LockedSkill
	InstalledPath string
	CreatedAt     string
}

func saveHistory(locked LockedSkill) error {
	if locked.InstalledPath == "" {
		return nil
	}
	if _, err := os.Stat(locked.InstalledPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	root, err := historyRoot(locked.DisplayIdentity())
	if err != nil {
		return err
	}
	dir := filepath.Join(root, time.Now().UTC().Format("20060102T150405.000000000Z"))
	filesDir := filepath.Join(dir, "files")
	if err := copyDir(locked.InstalledPath, filesDir); err != nil {
		return err
	}
	data, err := json.MarshalIndent(historyManifest{Locked: locked}, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0o644)
}

func latestHistory(identity string) (historySnapshot, error) {
	return historySnapshotForVersion(identity, "")
}

func historySnapshotForVersion(identity string, version string) (historySnapshot, error) {
	snapshots, err := historySnapshots(identity)
	if err != nil {
		return historySnapshot{}, err
	}
	if version == "" {
		return snapshots[len(snapshots)-1], nil
	}
	for i := len(snapshots) - 1; i >= 0; i-- {
		if snapshots[i].Locked.Version == version {
			return snapshots[i], nil
		}
	}
	return historySnapshot{}, fmt.Errorf("no rollback history for %s at version %s", identity, version)
}

func historySnapshots(identity string) ([]historySnapshot, error) {
	root, err := historyRoot(identity)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no rollback history for %s", identity)
		}
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("no rollback history for %s", identity)
	}
	sort.Strings(names)
	snapshots := make([]historySnapshot, 0, len(names))
	for _, name := range names {
		dir := filepath.Join(root, name)
		data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
		if err != nil {
			return nil, err
		}
		var manifest historyManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			return nil, err
		}
		snapshots = append(snapshots, historySnapshot{
			Locked:        manifest.Locked,
			InstalledPath: filepath.Join(dir, "files"),
			CreatedAt:     name,
		})
	}
	return snapshots, nil
}

func historyRoot(identity string) (string, error) {
	home, err := config.DefaultHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "history", skill.SafeIdentity(identity)), nil
}

func (l *LockFile) upsert(skill LockedSkill) {
	for i, existing := range l.Skills {
		if existing.displayIdentity() == skill.displayIdentity() {
			l.Skills[i] = skill
			return
		}
	}
	l.Skills = append(l.Skills, skill)
}

func (s LockedSkill) displayIdentity() string {
	return s.DisplayIdentity()
}

func (s LockedSkill) DisplayIdentity() string {
	if s.Identity != "" {
		return s.Identity
	}
	if s.Namespace != "" {
		return skill.Identity(s.Namespace, s.Name)
	}
	return s.Name
}

type resolvedSource struct {
	Path      string
	Type      string
	Registry  string
	URL       string
	Ref       string
	Commit    string
	Subpath   string
	CacheName string
}

func resolveSource(workDir string, cfg config.Config, spec string) (resolvedSource, error) {
	var sourceRef string
	spec, sourceRef = splitPinnedRef(spec)
	if looksLikePath(spec) {
		abs, err := absoluteFrom(workDir, spec)
		if err != nil {
			return resolvedSource{}, err
		}
		return resolvedSource{Path: abs, Type: "local", Ref: sourceRef}, nil
	}
	registryName, skillName, ok := strings.Cut(spec, "/")
	if !ok || registryName == "" || skillName == "" {
		return resolvedSource{}, fmt.Errorf("install spec must be a path or registry/skill")
	}
	reg, ok := cfg.Registries[registryName]
	if !ok {
		return resolvedSource{}, fmt.Errorf("unknown registry %q", registryName)
	}
	switch reg.Type {
	case "local":
		if indexed, ok, err := registry.ResolveIndexedSkill(reg.Path, skillName); err != nil {
			return resolvedSource{}, err
		} else if ok {
			return resolveIndexedSource(registryName, reg.Path, "", "", indexed, sourceRef)
		}
		return resolvedSource{
			Path:     filepath.Join(reg.Path, skillName),
			Type:     registry.SourceTypeRegistry,
			Registry: registryName,
			Ref:      sourceRef,
			Subpath:  filepath.ToSlash(skillName),
		}, nil
	case "git":
		if sourceRef != "" {
			cachePath, commit, err := registry.EnsureGitCacheAtRef(registryName, reg.URL, sourceRef)
			if err != nil {
				return resolvedSource{}, err
			}
			if indexed, ok, err := registry.ResolveIndexedSkill(cachePath, skillName); err != nil {
				return resolvedSource{}, err
			} else if ok {
				return resolveIndexedSource(registryName, cachePath, reg.URL, commit, indexed, sourceRef)
			}
			return resolvedSource{
				Path:      filepath.Join(cachePath, skillName),
				Type:      registry.SourceTypeGit,
				Registry:  registryName,
				URL:       reg.URL,
				Ref:       sourceRef,
				Commit:    commit,
				Subpath:   filepath.ToSlash(skillName),
				CacheName: registryName,
			}, nil
		}
		cachePath, err := registry.EnsureGitCache(registryName, reg.URL)
		if err != nil {
			return resolvedSource{}, err
		}
		commit, err := registry.GitCommit(cachePath)
		if err != nil {
			return resolvedSource{}, err
		}
		if indexed, ok, err := registry.ResolveIndexedSkill(cachePath, skillName); err != nil {
			return resolvedSource{}, err
		} else if ok {
			return resolveIndexedSource(registryName, cachePath, reg.URL, commit, indexed, "")
		}
		return resolvedSource{
			Path:      filepath.Join(cachePath, skillName),
			Type:      registry.SourceTypeGit,
			Registry:  registryName,
			URL:       reg.URL,
			Commit:    commit,
			Subpath:   filepath.ToSlash(skillName),
			CacheName: registryName,
		}, nil
	default:
		return resolvedSource{}, fmt.Errorf("unsupported registry type %q", reg.Type)
	}
}

func resolveIndexedSource(registryName string, root string, registryURL string, registryCommit string, indexed registry.IndexSkill, sourceRef string) (resolvedSource, error) {
	ref := sourceRef
	if ref == "" {
		ref = indexed.Source.Ref
	}
	switch indexed.Source.Type {
	case registry.SourceTypeRegistry:
		sourcePath, err := registry.RegistrySourcePath(root, indexed.Source.Path)
		if err != nil {
			return resolvedSource{}, err
		}
		sourceType := registry.SourceTypeRegistry
		sourceURL := ""
		sourceCommit := ""
		cacheName := ""
		if registryURL != "" {
			sourceType = registry.SourceTypeGit
			sourceURL = registryURL
			sourceCommit = registryCommit
			cacheName = registryName
		}
		return resolvedSource{
			Path:      sourcePath,
			Type:      sourceType,
			Registry:  registryName,
			URL:       sourceURL,
			Ref:       ref,
			Commit:    sourceCommit,
			Subpath:   indexed.Source.Path,
			CacheName: cacheName,
		}, nil
	case registry.SourceTypeGit:
		cacheName := registryName + "__" + skill.SafeIdentity(indexed.Identity)
		cachePath, commit, err := registry.EnsureGitCacheAtRef(cacheName, indexed.Source.URL, ref)
		if err != nil {
			return resolvedSource{}, err
		}
		sourcePath, err := registry.RegistrySourcePath(cachePath, indexed.Source.Path)
		if err != nil {
			return resolvedSource{}, err
		}
		return resolvedSource{
			Path:      sourcePath,
			Type:      registry.SourceTypeGit,
			Registry:  registryName,
			URL:       indexed.Source.URL,
			Ref:       ref,
			Commit:    commit,
			Subpath:   indexed.Source.Path,
			CacheName: cacheName,
		}, nil
	default:
		return resolvedSource{}, fmt.Errorf("unsupported catalog source type %q", indexed.Source.Type)
	}
}

func splitPinnedRef(spec string) (string, string) {
	if looksLikePath(spec) {
		return spec, ""
	}
	base, ref, ok := strings.Cut(spec, "@")
	if !ok {
		return spec, ""
	}
	return base, ref
}

func looksLikePath(spec string) bool {
	if spec == "." || strings.HasPrefix(spec, ".") || strings.HasPrefix(spec, "/") || strings.HasPrefix(spec, "~") {
		return true
	}
	// Recognize OS-native absolute paths, e.g. Windows "C:\foo" or "\\host\share".
	if filepath.IsAbs(spec) {
		return true
	}
	// On platforms where the separator is not "/" (Windows), a backslash never
	// appears in a registry/skill spec (those always use "/"), so any spec
	// containing the native separator is a filesystem path.
	if os.PathSeparator != '/' && strings.ContainsRune(spec, os.PathSeparator) {
		return true
	}
	return false
}

func absoluteFrom(workDir string, path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}
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
		return copyFile(path, target, info.Mode())
	})
}

func copyFile(src string, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
