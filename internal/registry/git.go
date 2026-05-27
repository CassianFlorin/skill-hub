package registry

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/cassian/skill-hub/internal/config"
)

func GitCachePath(name string) (string, error) {
	home, err := config.DefaultHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "cache", "registries", name), nil
}

func EnsureGitCache(name string, url string) (string, error) {
	cachePath, err := GitCachePath(name)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(filepath.Join(cachePath, ".git")); err == nil {
		if err := git(cachePath, "pull", "--ff-only"); err != nil {
			return "", err
		}
		return cachePath, nil
	}
	if err := os.RemoveAll(cachePath); err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		return "", err
	}
	if err := git("", "clone", url, cachePath); err != nil {
		return "", err
	}
	return cachePath, nil
}

func git(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %v failed: %w\n%s", args, err, string(output))
	}
	return nil
}
