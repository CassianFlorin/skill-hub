package deploy

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cassian/skill-hub/internal/install"
	"github.com/cassian/skill-hub/internal/skill"
)

func CodexDir() (string, error) {
	return RuntimeDir("codex")
}

func ClaudeDir() (string, error) {
	return RuntimeDir("claude")
}

type Options struct {
	Identity string
	DryRun   bool
	Force    bool
}

type Runtime struct {
	Name          string
	EnvVar        string
	DefaultSubdir string
}

type Result struct {
	Identity string
	Runtime  string
	State    string
	Reason   string
	DryRun   bool
}

type Status struct {
	Identity string
	Runtime  string
	State    string
}

const (
	StateDeployed    = "deployed"
	StateWouldDeploy = "would-deploy"
	StateSkipped     = "skipped"
	StateConflict    = "conflict"
	StateMissing     = "missing"
	StateDrifted     = "drifted"
	StateUnsupported = "unsupported"
)

var supportedRuntimes = []Runtime{
	{Name: "codex", EnvVar: "SKILLHUB_CODEX_DIR", DefaultSubdir: filepath.Join(".codex", "skills")},
	{Name: "claude", EnvVar: "SKILLHUB_CLAUDE_DIR", DefaultSubdir: filepath.Join(".claude", "skills")},
}

func SupportedRuntimes() []Runtime {
	runtimes := make([]Runtime, len(supportedRuntimes))
	copy(runtimes, supportedRuntimes)
	return runtimes
}

func RuntimeNames() []string {
	names := make([]string, 0, len(supportedRuntimes))
	for _, runtime := range supportedRuntimes {
		names = append(names, runtime.Name)
	}
	return names
}

func DeployCodex(options Options) ([]Result, error) {
	return deployRuntime("codex", options)
}

func DeployClaude(options Options) ([]Result, error) {
	return deployRuntime("claude", options)
}

func Statuses(runtime string) ([]Status, error) {
	lock, err := install.LoadLock()
	if err != nil {
		return nil, err
	}
	runtimes, err := selectedRuntimeNames(runtime)
	if err != nil {
		return nil, err
	}
	var statuses []Status
	for _, locked := range lock.Skills {
		for _, runtime := range runtimes {
			state := StateUnsupported
			if supportsRuntime(locked, runtime) {
				targetRoot, err := RuntimeDir(runtime)
				if err != nil {
					return nil, err
				}
				target := filepath.Join(targetRoot, skill.SafeIdentity(locked.DisplayIdentity()))
				state = StateMissing
				if _, err := os.Stat(target); err == nil {
					checksum, err := skill.ChecksumDir(target)
					if err != nil {
						return nil, err
					}
					if checksum == locked.Checksum {
						state = StateDeployed
					} else {
						state = StateDrifted
					}
				} else if err != nil && !os.IsNotExist(err) {
					return nil, err
				}
			}
			statuses = append(statuses, Status{
				Identity: locked.DisplayIdentity(),
				Runtime:  runtime,
				State:    state,
			})
		}
	}
	return statuses, nil
}

func selectedRuntimeNames(runtime string) ([]string, error) {
	if runtime == "" {
		return RuntimeNames(), nil
	}
	if _, ok := runtimeByName(runtime); !ok {
		return nil, fmt.Errorf("unsupported runtime %q", runtime)
	}
	return []string{runtime}, nil
}

func RuntimeDir(name string) (string, error) {
	runtime, ok := runtimeByName(name)
	if !ok {
		return "", fmt.Errorf("unsupported runtime %q", name)
	}
	if dir := os.Getenv(runtime.EnvVar); dir != "" {
		return filepath.Abs(dir)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, runtime.DefaultSubdir), nil
}

func runtimeByName(name string) (Runtime, bool) {
	for _, runtime := range supportedRuntimes {
		if runtime.Name == name {
			return runtime, true
		}
	}
	return Runtime{}, false
}

func RuntimeTarget(runtime string, identity string) (string, error) {
	root, err := RuntimeDir(runtime)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, skill.SafeIdentity(identity)), nil
}

func deployRuntime(runtime string, options Options) ([]Result, error) {
	targetRoot, err := RuntimeDir(runtime)
	if err != nil {
		return nil, err
	}
	lock, err := install.LoadLock()
	if err != nil {
		return nil, err
	}
	type candidate struct {
		Index  int
		Locked install.LockedSkill
		Target string
	}
	var candidates []candidate
	var results []Result
	for i, locked := range lock.Skills {
		identity := locked.DisplayIdentity()
		if options.Identity != "" && options.Identity != identity && options.Identity != locked.Name {
			continue
		}
		if !supportsRuntime(locked, runtime) {
			if options.Identity != "" {
				return nil, fmt.Errorf("%s does not support runtime %q", identity, runtime)
			}
			results = append(results, Result{
				Identity: identity,
				Runtime:  runtime,
				State:    StateSkipped,
				Reason:   "unsupported",
			})
			continue
		}
		candidates = append(candidates, candidate{
			Index:  i,
			Locked: locked,
			Target: filepath.Join(targetRoot, skill.SafeIdentity(identity)),
		})
	}
	if options.Identity != "" && len(candidates) == 0 {
		return nil, fmt.Errorf("unknown installed skill %q", options.Identity)
	}

	var conflicts []Result
	for _, candidate := range candidates {
		identity := candidate.Locked.DisplayIdentity()
		if _, err := os.Stat(candidate.Target); err == nil && !options.Force {
			conflict := Result{
				Identity: identity,
				Runtime:  runtime,
				State:    StateConflict,
				Reason:   "target already exists: " + candidate.Target,
			}
			conflicts = append(conflicts, conflict)
			continue
		} else if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
	}

	if options.DryRun {
		results = append(results, conflicts...)
		for _, candidate := range candidates {
			identity := candidate.Locked.DisplayIdentity()
			if hasConflict(conflicts, identity) {
				continue
			}
			results = append(results, Result{
				Identity: identity,
				Runtime:  runtime,
				State:    StateWouldDeploy,
				DryRun:   true,
			})
		}
		return results, nil
	}

	if len(conflicts) > 0 {
		first := conflicts[0]
		return nil, fmt.Errorf("deploy %s: %s", first.Identity, first.Reason)
	}

	for _, candidate := range candidates {
		identity := candidate.Locked.DisplayIdentity()
		if err := copyDir(candidate.Locked.InstalledPath, candidate.Target); err != nil {
			return nil, fmt.Errorf("deploy %s: %w", identity, err)
		}
		lock.Skills[candidate.Index].DeployedTo = appendRuntime(lock.Skills[candidate.Index].DeployedTo, runtime)
		results = append(results, Result{Identity: identity, Runtime: runtime, State: StateDeployed})
	}
	if err := install.SaveLock(lock); err != nil {
		return nil, err
	}
	return results, nil
}

func supportsRuntime(locked install.LockedSkill, runtime string) bool {
	if len(locked.Targets) == 0 {
		return true
	}
	for _, target := range locked.Targets {
		if target == runtime {
			return true
		}
	}
	return false
}

func hasConflict(conflicts []Result, identity string) bool {
	for _, conflict := range conflicts {
		if conflict.Identity == identity {
			return true
		}
	}
	return false
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
