package cli

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/cassian/skill-hub/internal/config"
	"github.com/cassian/skill-hub/internal/registry"
)

func runRegistry(args []string, stdout io.Writer, workDir string) error {
	if len(args) < 1 {
		return errors.New(registryUsage)
	}
	switch args[0] {
	case "add":
		return runRegistryAdd(args[1:], stdout, workDir)
	case "list":
		return runRegistryList(stdout, workDir)
	case "sync":
		return runRegistrySync(args[1:], stdout, workDir)
	case "index":
		return runRegistryIndex(args[1:], stdout, workDir)
	default:
		return errors.New(registryUsage)
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
		return withCLIHint(fmt.Errorf("unknown registry %q", name))
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

func runRegistryList(stdout io.Writer, workDir string) error {
	cfg, err := config.Load(workDir)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprint(stdout, formatRegistryList(registry.ListRegistries(cfg)))
	return nil
}

func runRegistrySync(args []string, stdout io.Writer, workDir string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: skillhub registry sync <registry|--all>")
	}
	cfg, err := config.Load(workDir)
	if err != nil {
		return err
	}
	if args[0] == "--all" {
		for _, status := range registry.ListRegistries(cfg) {
			reg := cfg.Registries[status.Name]
			count, err := registry.SyncRegistry(status.Name, reg)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(stdout, "synced %s with %d skills\n", status.Name, count)
		}
		return nil
	}
	reg, ok := cfg.Registries[args[0]]
	if !ok {
		return withCLIHint(fmt.Errorf("unknown registry %q", args[0]))
	}
	count, err := registry.SyncRegistry(args[0], reg)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "synced %s with %d skills\n", args[0], count)
	return nil
}
