package install

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cassian/skill-hub/internal/config"
	"github.com/cassian/skill-hub/internal/skill"
)

const LockFileName = "skillhub.lock"

type LockFile struct {
	Skills []LockedSkill `json:"skills"`
}

type LockedSkill struct {
	Name           string   `json:"name"`
	Version        string   `json:"version"`
	Description    string   `json:"description,omitempty"`
	SourceType     string   `json:"source_type"`
	SourceRegistry string   `json:"source_registry,omitempty"`
	SourcePath     string   `json:"source_path"`
	InstalledPath  string   `json:"installed_path"`
	Targets        []string `json:"targets,omitempty"`
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
	sourcePath, sourceType, sourceRegistry, err := resolveSource(workDir, cfg, spec)
	if err != nil {
		return LockedSkill{}, err
	}
	meta, err := skill.LoadMetadata(sourcePath)
	if err != nil {
		return LockedSkill{}, err
	}
	installedPath := filepath.Join(cfg.InstallDir, meta.Name)
	if err := copyDir(sourcePath, installedPath); err != nil {
		return LockedSkill{}, err
	}
	locked := LockedSkill{
		Name:           meta.Name,
		Version:        meta.Version,
		Description:    meta.Description,
		SourceType:     sourceType,
		SourceRegistry: sourceRegistry,
		SourcePath:     sourcePath,
		InstalledPath:  installedPath,
		Targets:        meta.Targets,
		UpdatedAt:      time.Now().UTC().Format(time.RFC3339),
	}
	lock, err := LoadLock()
	if err != nil {
		return LockedSkill{}, err
	}
	lock.upsert(locked)
	if err := SaveLock(lock); err != nil {
		return LockedSkill{}, err
	}
	return locked, nil
}

func UpdateAll() ([][3]string, error) {
	lock, err := LoadLock()
	if err != nil {
		return nil, err
	}
	var changes [][3]string
	for i, locked := range lock.Skills {
		meta, err := skill.LoadMetadata(locked.SourcePath)
		if err != nil {
			return nil, fmt.Errorf("update %s: %w", locked.Name, err)
		}
		if meta.Version == locked.Version {
			continue
		}
		if err := copyDir(locked.SourcePath, locked.InstalledPath); err != nil {
			return nil, err
		}
		changes = append(changes, [3]string{locked.Name, locked.Version, meta.Version})
		lock.Skills[i].Version = meta.Version
		lock.Skills[i].Description = meta.Description
		lock.Skills[i].Targets = meta.Targets
		lock.Skills[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if err := SaveLock(lock); err != nil {
		return nil, err
	}
	return changes, nil
}

func (l *LockFile) upsert(skill LockedSkill) {
	for i, existing := range l.Skills {
		if existing.Name == skill.Name {
			l.Skills[i] = skill
			return
		}
	}
	l.Skills = append(l.Skills, skill)
}

func resolveSource(workDir string, cfg config.Config, spec string) (path string, sourceType string, registry string, err error) {
	if looksLikePath(spec) {
		abs, err := absoluteFrom(workDir, spec)
		if err != nil {
			return "", "", "", err
		}
		return abs, "local", "", nil
	}
	registryName, skillName, ok := strings.Cut(spec, "/")
	if !ok || registryName == "" || skillName == "" {
		return "", "", "", fmt.Errorf("install spec must be a path or registry/skill")
	}
	reg, ok := cfg.Registries[registryName]
	if !ok {
		return "", "", "", fmt.Errorf("unknown registry %q", registryName)
	}
	switch reg.Type {
	case "local":
		return filepath.Join(reg.Path, skillName), "registry", registryName, nil
	case "git":
		return "", "", "", fmt.Errorf("git registry install requires a local checkout in this MVP")
	default:
		return "", "", "", fmt.Errorf("unsupported registry type %q", reg.Type)
	}
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
