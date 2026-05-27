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
	Identity       string   `json:"identity"`
	Name           string   `json:"name"`
	Namespace      string   `json:"namespace"`
	Version        string   `json:"version"`
	Description    string   `json:"description,omitempty"`
	SourceType     string   `json:"source_type"`
	SourceRegistry string   `json:"source_registry,omitempty"`
	SourceURL      string   `json:"source_url,omitempty"`
	SourceRef      string   `json:"source_ref,omitempty"`
	SourceCommit   string   `json:"source_commit,omitempty"`
	SourcePath     string   `json:"source_path"`
	Checksum       string   `json:"checksum,omitempty"`
	InstalledPath  string   `json:"installed_path"`
	Targets        []string `json:"targets,omitempty"`
	DeployedTo     []string `json:"deployed_runtimes,omitempty"`
	UpdatedAt      string   `json:"updated_at"`
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
	sourcePath, sourceType, sourceRegistry, sourceURL, sourceRef, sourceCommit, err := resolveSource(workDir, cfg, spec)
	if err != nil {
		return LockedSkill{}, err
	}
	meta, err := skill.LoadCompatibleMetadata(sourcePath, sourceRegistry)
	if err != nil {
		return LockedSkill{}, err
	}
	identity := skill.Identity(meta.Namespace, meta.Name)
	if sourceRef != "" && sourceType != "git" && meta.Version != sourceRef {
		return LockedSkill{}, fmt.Errorf("version %s not available for %s", sourceRef, identity)
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
	if err := copyDir(sourcePath, installedPath); err != nil {
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
		SourceType:     sourceType,
		SourceRegistry: sourceRegistry,
		SourceURL:      sourceURL,
		SourceRef:      sourceRef,
		SourceCommit:   sourceCommit,
		SourcePath:     sourcePath,
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
	lock, err := LoadLock()
	if err != nil {
		return LockedSkill{}, err
	}
	current, ok := lock.find(identity)
	if !ok {
		return LockedSkill{}, fmt.Errorf("unknown installed skill %q", identity)
	}
	snapshot, err := latestHistory(current.DisplayIdentity())
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

func UpdateAll() ([][3]string, error) {
	lock, err := LoadLock()
	if err != nil {
		return nil, err
	}
	var changes [][3]string
	for i, locked := range lock.Skills {
		if locked.SourceType == "git" {
			cachePath, err := registry.EnsureGitCache(locked.SourceRegistry, locked.SourceURL)
			if err != nil {
				return nil, fmt.Errorf("update %s: %w", locked.Name, err)
			}
			locked.SourcePath = filepath.Join(cachePath, locked.Name)
			lock.Skills[i].SourcePath = locked.SourcePath
		}
		meta, err := skill.LoadCompatibleMetadata(locked.SourcePath, locked.SourceRegistry)
		if err != nil {
			return nil, fmt.Errorf("update %s: %w", locked.Name, err)
		}
		if meta.Version == locked.Version {
			continue
		}
		if err := copyDir(locked.SourcePath, locked.InstalledPath); err != nil {
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
		changes = append(changes, [3]string{locked.displayIdentity(), locked.Version, meta.Version})
		lock.Skills[i].Identity = identity
		lock.Skills[i].Name = meta.Name
		lock.Skills[i].Namespace = meta.Namespace
		lock.Skills[i].Version = meta.Version
		lock.Skills[i].Description = meta.Description
		lock.Skills[i].Targets = meta.Targets
		lock.Skills[i].Checksum = checksum
		lock.Skills[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if err := SaveLock(lock); err != nil {
		return nil, err
	}
	return changes, nil
}

type historyManifest struct {
	Locked LockedSkill `json:"locked"`
}

type historySnapshot struct {
	Locked        LockedSkill
	InstalledPath string
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
	root, err := historyRoot(identity)
	if err != nil {
		return historySnapshot{}, err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return historySnapshot{}, fmt.Errorf("no rollback history for %s", identity)
		}
		return historySnapshot{}, err
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	if len(names) == 0 {
		return historySnapshot{}, fmt.Errorf("no rollback history for %s", identity)
	}
	sort.Strings(names)
	dir := filepath.Join(root, names[len(names)-1])
	data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return historySnapshot{}, err
	}
	var manifest historyManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return historySnapshot{}, err
	}
	return historySnapshot{
		Locked:        manifest.Locked,
		InstalledPath: filepath.Join(dir, "files"),
	}, nil
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

func resolveSource(workDir string, cfg config.Config, spec string) (path string, sourceType string, registryName string, sourceURL string, sourceRef string, sourceCommit string, err error) {
	spec, sourceRef = splitPinnedRef(spec)
	if looksLikePath(spec) {
		abs, err := absoluteFrom(workDir, spec)
		if err != nil {
			return "", "", "", "", "", "", err
		}
		return abs, "local", "", "", sourceRef, "", nil
	}
	registryName, skillName, ok := strings.Cut(spec, "/")
	if !ok || registryName == "" || skillName == "" {
		return "", "", "", "", "", "", fmt.Errorf("install spec must be a path or registry/skill")
	}
	reg, ok := cfg.Registries[registryName]
	if !ok {
		return "", "", "", "", "", "", fmt.Errorf("unknown registry %q", registryName)
	}
	switch reg.Type {
	case "local":
		if indexedPath, ok, err := registry.ResolveIndexedPath(reg.Path, skillName); err != nil {
			return "", "", "", "", "", "", err
		} else if ok {
			return indexedPath, "registry", registryName, "", sourceRef, "", nil
		}
		return filepath.Join(reg.Path, skillName), "registry", registryName, "", sourceRef, "", nil
	case "git":
		cachePath, commit, err := registry.EnsureGitCacheAtRef(registryName, reg.URL, sourceRef)
		if err != nil {
			return "", "", "", "", "", "", err
		}
		if indexedPath, ok, err := registry.ResolveIndexedPath(cachePath, skillName); err != nil {
			return "", "", "", "", "", "", err
		} else if ok {
			return indexedPath, "git", registryName, reg.URL, sourceRef, commit, nil
		}
		return filepath.Join(cachePath, skillName), "git", registryName, reg.URL, sourceRef, commit, nil
	default:
		return "", "", "", "", "", "", fmt.Errorf("unsupported registry type %q", reg.Type)
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
	return spec == "." || strings.HasPrefix(spec, ".") || strings.HasPrefix(spec, "/") || strings.HasPrefix(spec, "~")
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
