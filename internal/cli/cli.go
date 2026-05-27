package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

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
	case "search":
		return runSearch(args[1:], stdout, workDir)
	case "info":
		return runInfo(args[1:], stdout, workDir)
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
	case "rollback":
		return runRollback(args[1:], stdout)
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

func runRollback(args []string, stdout io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: skillhub rollback <identity>")
	}
	locked, err := install.Rollback(args[0])
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "rolled back %s to %s\n", locked.DisplayIdentity(), locked.Version)
	return nil
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
	if len(args) != 2 {
		return fmt.Errorf("usage: skillhub registry index <generate|validate> <registry>")
	}
	action, name := args[0], args[1]
	cfg, err := config.Load(workDir)
	if err != nil {
		return err
	}
	reg, ok := cfg.Registries[name]
	if !ok {
		return fmt.Errorf("unknown registry %q", name)
	}
	switch action {
	case "generate":
		index, _, err := registry.GenerateIndex(name, reg)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(stdout, "indexed %s with %d skills\n", name, len(index.Skills))
	case "validate":
		count, err := registry.ValidateIndex(name, reg)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(stdout, "validated %s with %d skills\n", name, count)
	default:
		return fmt.Errorf("usage: skillhub registry index <generate|validate> <registry>")
	}
	return nil
}

func runSearch(args []string, stdout io.Writer, workDir string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: skillhub search <query>")
	}
	cfg, err := config.Load(workDir)
	if err != nil {
		return err
	}
	results, err := registry.SearchIndexes(cfg, args[0])
	if err != nil {
		return err
	}
	if len(results) == 0 {
		_, _ = fmt.Fprintln(stdout, "no skills found")
		return nil
	}
	for _, result := range results {
		_, _ = fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\n", result.Registry, result.Skill.Identity, result.Skill.Version, result.Skill.Description)
	}
	return nil
}

func runInfo(args []string, stdout io.Writer, workDir string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: skillhub info <registry/identity|identity>")
	}
	cfg, err := config.Load(workDir)
	if err != nil {
		return err
	}
	result, ok, err := registry.FindIndexedSkill(cfg, args[0])
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("skill %q not found", args[0])
	}
	indexed := result.Skill
	_, _ = fmt.Fprintf(stdout, "identity: %s\n", indexed.Identity)
	_, _ = fmt.Fprintf(stdout, "registry: %s\n", result.Registry)
	_, _ = fmt.Fprintf(stdout, "name: %s\n", indexed.Name)
	_, _ = fmt.Fprintf(stdout, "namespace: %s\n", indexed.Namespace)
	_, _ = fmt.Fprintf(stdout, "version: %s\n", indexed.Version)
	_, _ = fmt.Fprintf(stdout, "description: %s\n", indexed.Description)
	_, _ = fmt.Fprintf(stdout, "targets: %s\n", strings.Join(indexed.Targets, ", "))
	_, _ = fmt.Fprintf(stdout, "tags: %s\n", strings.Join(indexed.Tags, ", "))
	_, _ = fmt.Fprintf(stdout, "source_type: %s\n", indexed.SourceType)
	_, _ = fmt.Fprintf(stdout, "source_path: %s\n", indexed.SourcePath)
	_, _ = fmt.Fprintf(stdout, "checksum: %s\n", indexed.Checksum)
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
	_, _ = fmt.Fprintln(stderr, "usage: skillhub <init|registry|search|info|install|rollback|list|update|deploy>")
	return fmt.Errorf("invalid command")
}
