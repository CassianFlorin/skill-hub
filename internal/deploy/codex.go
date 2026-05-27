package deploy

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cassian/skill-hub/internal/install"
)

func CodexDir() (string, error) {
	if dir := os.Getenv("SKILLHUB_CODEX_DIR"); dir != "" {
		return filepath.Abs(dir)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex", "skills"), nil
}

func DeployCodex() ([]string, error) {
	lock, err := install.LoadLock()
	if err != nil {
		return nil, err
	}
	targetRoot, err := CodexDir()
	if err != nil {
		return nil, err
	}
	var deployed []string
	for _, locked := range lock.Skills {
		target := filepath.Join(targetRoot, locked.Name)
		if err := copyDir(locked.InstalledPath, target); err != nil {
			return nil, fmt.Errorf("deploy %s: %w", locked.Name, err)
		}
		deployed = append(deployed, locked.Name)
	}
	return deployed, nil
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
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
}
