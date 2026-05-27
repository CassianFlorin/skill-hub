package deploy

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cassian/skill-hub/internal/install"
	"github.com/cassian/skill-hub/internal/skill"
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

func ClaudeDir() (string, error) {
	if dir := os.Getenv("SKILLHUB_CLAUDE_DIR"); dir != "" {
		return filepath.Abs(dir)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "skills"), nil
}

type Options struct {
	Identity string
	DryRun   bool
	Force    bool
}

type Result struct {
	Identity string
	DryRun   bool
}

func DeployCodex(options Options) ([]Result, error) {
	targetRoot, err := CodexDir()
	if err != nil {
		return nil, err
	}
	return deployRuntime("codex", targetRoot, options)
}

func DeployClaude(options Options) ([]Result, error) {
	targetRoot, err := ClaudeDir()
	if err != nil {
		return nil, err
	}
	return deployRuntime("claude", targetRoot, options)
}

func deployRuntime(runtime string, targetRoot string, options Options) ([]Result, error) {
	lock, err := install.LoadLock()
	if err != nil {
		return nil, err
	}
	var deployed []Result
	for i, locked := range lock.Skills {
		identity := locked.DisplayIdentity()
		if options.Identity != "" && options.Identity != identity && options.Identity != locked.Name {
			continue
		}
		target := filepath.Join(targetRoot, skill.SafeIdentity(identity))
		if options.DryRun {
			deployed = append(deployed, Result{Identity: identity, DryRun: true})
			continue
		}
		if _, err := os.Stat(target); err == nil && !options.Force {
			return nil, fmt.Errorf("deploy %s: target already exists: %s", identity, target)
		} else if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		if err := copyDir(locked.InstalledPath, target); err != nil {
			return nil, fmt.Errorf("deploy %s: %w", identity, err)
		}
		lock.Skills[i].DeployedTo = appendRuntime(lock.Skills[i].DeployedTo, runtime)
		deployed = append(deployed, Result{Identity: identity})
	}
	if options.Identity != "" && len(deployed) == 0 {
		return nil, fmt.Errorf("unknown installed skill %q", options.Identity)
	}
	if !options.DryRun {
		if err := install.SaveLock(lock); err != nil {
			return nil, err
		}
	}
	return deployed, nil
}

func appendRuntime(runtimes []string, runtime string) []string {
	for _, existing := range runtimes {
		if existing == runtime {
			return runtimes
		}
	}
	return append(runtimes, runtime)
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
