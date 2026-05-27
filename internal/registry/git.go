package registry

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cassian/skill-hub/internal/config"
	"github.com/cassian/skill-hub/internal/skill"
)

func GitCachePath(name string) (string, error) {
	return GitRefCachePath(name, "")
}

func GitRefCachePath(name string, ref string) (string, error) {
	home, err := config.DefaultHome()
	if err != nil {
		return "", err
	}
	if ref == "" {
		return filepath.Join(home, "cache", "registries", name), nil
	}
	return filepath.Join(home, "cache", "registries", name+"__"+skill.SafeIdentity(ref)), nil
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

func EnsureGitCacheAtRef(name string, url string, ref string) (string, string, error) {
	if ref == "" {
		cachePath, err := EnsureGitCache(name, url)
		if err != nil {
			return "", "", err
		}
		commit, err := GitCommit(cachePath)
		return cachePath, commit, err
	}
	cachePath, err := GitRefCachePath(name, ref)
	if err != nil {
		return "", "", err
	}
	if _, err := os.Stat(filepath.Join(cachePath, ".git")); err != nil {
		if err := os.RemoveAll(cachePath); err != nil {
			return "", "", err
		}
		if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
			return "", "", err
		}
		if err := git("", "clone", url, cachePath); err != nil {
			return "", "", err
		}
	} else {
		if err := git(cachePath, "fetch", "--tags", "--prune"); err != nil {
			return "", "", err
		}
	}
	if err := git(cachePath, "checkout", "--detach", ref); err != nil {
		return "", "", err
	}
	commit, err := GitCommit(cachePath)
	if err != nil {
		return "", "", err
	}
	return cachePath, commit, nil
}

func GitCommit(path string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = path
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD failed: %w\n%s", err, string(output))
	}
	return strings.TrimSpace(string(output)), nil
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
