package cli

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/cassian/skill-hub/internal/config"
	"github.com/cassian/skill-hub/internal/deploy"
	"github.com/cassian/skill-hub/internal/install"
	"github.com/cassian/skill-hub/internal/registry"
)

func Run(args []string, stdout io.Writer, stderr io.Writer, workDir string) error {
	if len(args) == 0 {
		return usage(stderr)
	}
	switch args[0] {
	case "init":
		return runInit(stdout, workDir)
	case "registry":
		return runRegistry(args[1:], stdout, workDir)
	case "install":
		if len(args) != 2 {
			return fmt.Errorf("usage: skillhub install <path|registry/skill>")
		}
		locked, err := install.Install(workDir, args[1])
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(stdout, "installed %s@%s\n", locked.DisplayIdentity(), locked.Version)
		return nil
	case "list":
		return runList(stdout)
	case "update":
		return runUpdate(stdout)
	case "deploy":
		return runDeploy(args[1:], stdout)
	default:
		return usage(stderr)
	}
}

func runInit(stdout io.Writer, workDir string) error {
	cfg, err := config.NewDefault()
	if err != nil {
		return err
	}
	if err := config.Save(workDir, cfg); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "initialized %s\n", config.Path(workDir))
	return nil
}

func runRegistry(args []string, stdout io.Writer, workDir string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: skillhub registry <add|index>")
	}
	switch args[0] {
	case "add":
		return runRegistryAdd(args[1:], stdout, workDir)
	case "index":
		return runRegistryIndex(args[1:], stdout, workDir)
	default:
		return fmt.Errorf("usage: skillhub registry <add|index>")
	}
}

func runRegistryAdd(args []string, stdout io.Writer, workDir string) error {
	if len(args) != 3 {
		return fmt.Errorf("usage: skillhub registry add <local|git> <name> <path-or-url>")
	}
	registryType, name, location := args[0], args[1], args[2]
	cfg, err := config.Load(workDir)
	if err != nil {
		return err
	}
	switch registryType {
	case "local":
		if !filepath.IsAbs(location) {
			location = filepath.Join(workDir, location)
		}
		abs, err := filepath.Abs(location)
		if err != nil {
			return err
		}
		cfg.Registries[name] = config.Registry{Type: "local", Path: abs}
	case "git":
		cfg.Registries[name] = config.Registry{Type: "git", URL: location}
	default:
		return fmt.Errorf("unsupported registry type %q", registryType)
	}
	if err := config.Save(workDir, cfg); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "registered %s\n", name)
	return nil
}

func runRegistryIndex(args []string, stdout io.Writer, workDir string) error {
	if len(args) != 2 || args[0] != "generate" {
		return fmt.Errorf("usage: skillhub registry index generate <registry>")
	}
	name := args[1]
	cfg, err := config.Load(workDir)
	if err != nil {
		return err
	}
	reg, ok := cfg.Registries[name]
	if !ok {
		return fmt.Errorf("unknown registry %q", name)
	}
	index, _, err := registry.GenerateIndex(name, reg)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "indexed %s with %d skills\n", name, len(index.Skills))
	return nil
}

func runList(stdout io.Writer) error {
	lock, err := install.LoadLock()
	if err != nil {
		return err
	}
	if len(lock.Skills) == 0 {
		_, _ = fmt.Fprintln(stdout, "no skills installed")
		return nil
	}
	for _, locked := range lock.Skills {
		_, _ = fmt.Fprintf(stdout, "%s\t%s\t%s\n", locked.DisplayIdentity(), locked.Version, locked.Description)
	}
	return nil
}

func runUpdate(stdout io.Writer) error {
	changes, err := install.UpdateAll()
	if err != nil {
		return err
	}
	if len(changes) == 0 {
		_, _ = fmt.Fprintln(stdout, "all skills are current")
		return nil
	}
	for _, change := range changes {
		_, _ = fmt.Fprintf(stdout, "updated %s %s -> %s\n", change[0], change[1], change[2])
	}
	return nil
}

func runDeploy(args []string, stdout io.Writer) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: skillhub deploy <codex|claude> [identity] [--dry-run] [--force]")
	}
	runtime := args[0]
	options := deploy.Options{}
	for _, arg := range args[1:] {
		switch arg {
		case "--dry-run":
			options.DryRun = true
		case "--force":
			options.Force = true
		default:
			if options.Identity != "" {
				return fmt.Errorf("usage: skillhub deploy codex [identity] [--dry-run] [--force]")
			}
			options.Identity = arg
		}
	}
	var deployed []deploy.Result
	var err error
	switch runtime {
	case "codex":
		deployed, err = deploy.DeployCodex(options)
	case "claude":
		deployed, err = deploy.DeployClaude(options)
	default:
		return fmt.Errorf("usage: skillhub deploy <codex|claude> [identity] [--dry-run] [--force]")
	}
	if err != nil {
		return err
	}
	for _, result := range deployed {
		if result.DryRun {
			_, _ = fmt.Fprintf(stdout, "would deploy %s to %s\n", result.Identity, runtime)
			continue
		}
		_, _ = fmt.Fprintf(stdout, "deployed %s to %s\n", result.Identity, runtime)
	}
	return nil
}

func usage(stderr io.Writer) error {
	_, _ = fmt.Fprintln(stderr, "usage: skillhub <init|registry|install|list|update|deploy>")
	return fmt.Errorf("invalid command")
}
