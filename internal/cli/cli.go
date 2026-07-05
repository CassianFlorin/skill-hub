package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/CassianFlorin/skill-hub/internal/config"
	"github.com/CassianFlorin/skill-hub/internal/deploy"
	"github.com/CassianFlorin/skill-hub/internal/install"
	"github.com/CassianFlorin/skill-hub/internal/registry"
	"github.com/CassianFlorin/skill-hub/internal/tui"
)

var Version = "dev"

func Run(args []string, stdout io.Writer, stderr io.Writer, workDir string) error {
	install.SkillhubVersion = Version
	if len(args) == 0 {
		return usage(stderr)
	}
	switch args[0] {
	case "help":
		return runHelp(args[1:], stdout)
	case "version":
		return runVersion(stdout)
	case "doctor":
		return runDoctor(args[1:], stdout, workDir)
	case "init":
		return runInit(stdout, workDir)
	case "registry":
		return runRegistry(args[1:], stdout, workDir)
	case "catalog":
		return runCatalog(args[1:], stdout, workDir)
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
			return withCLIHint(err)
		}
		_, _ = fmt.Fprintf(stdout, "installed %s@%s\n", locked.DisplayIdentity(), locked.Version)
		return nil
	case "publish":
		return runPublish(args[1:], stdout, workDir)
	case "rollback":
		return runRollback(args[1:], stdout)
	case "history":
		return runHistory(args[1:], stdout)
	case "hold":
		return runHold(args[1:], stdout)
	case "unhold":
		return runUnhold(args[1:], stdout)
	case "holds":
		return runHolds(args[1:], stdout)
	case "uninstall":
		return runUninstall(args[1:], stdout)
	case "list":
		return runList(args[1:], stdout, workDir)
	case "check":
		return runCheck(args[1:], stdout)
	case "update":
		return runUpdate(args[1:], stdout)
	case "deploy":
		return runDeploy(args[1:], stdout)
	case "audit":
		return runAudit(args[1:], stdout)
	case "tui":
		if len(args) != 1 {
			return fmt.Errorf("usage: skillhub tui")
		}
		return tui.Run(workDir)
	default:
		return usage(stderr)
	}
}

func runVersion(stdout io.Writer) error {
	_, _ = fmt.Fprintf(stdout, "skillhub %s\n", Version)
	return nil
}

const rootUsage = "usage: skillhub <command>"
const registryUsage = "usage: skillhub registry <add|list|sync|index>"
const listUsage = "usage: skillhub list [--scope all|global|project] [--json]"
const catalogUsage = "usage: skillhub catalog <list|featured|tags|targets|namespaces|trust|export>"

func runDoctor(args []string, stdout io.Writer, workDir string) error {
	jsonOutput := false
	for _, arg := range args {
		if arg != "--json" {
			return fmt.Errorf("usage: skillhub doctor [--json]")
		}
		jsonOutput = true
	}
	cfg, err := config.Load(workDir)
	if err != nil {
		return err
	}
	home, err := config.DefaultHome()
	if err != nil {
		return err
	}
	runtimes := make([]doctorRuntimeJSON, 0)
	for _, runtime := range deploy.SupportedRuntimes() {
		dir, err := deploy.RuntimeDir(runtime.Name)
		if err != nil {
			return err
		}
		runtimes = append(runtimes, doctorRuntimeJSON{Name: runtime.Name, Dir: dir})
	}
	statuses := registry.ListRegistries(cfg)
	lock, err := install.LoadLock()
	if err != nil {
		return err
	}
	if jsonOutput {
		return writeJSON(stdout, doctorJSON{
			Config:     "ok",
			ConfigPath: config.Path(workDir),
			Home:       home,
			InstallDir: cfg.InstallDir,
			Runtimes:   runtimes,
			Registries: registryJSONList(statuses),
			Installed:  len(lock.Skills),
		})
	}
	_, _ = fmt.Fprintln(stdout, "config: ok")
	_, _ = fmt.Fprintf(stdout, "config.path: %s\n", config.Path(workDir))
	_, _ = fmt.Fprintf(stdout, "home: %s\n", home)
	_, _ = fmt.Fprintf(stdout, "install_dir: %s\n", cfg.InstallDir)
	for _, runtime := range runtimes {
		_, _ = fmt.Fprintf(stdout, "runtime %s: %s\n", runtime.Name, runtime.Dir)
	}
	_, _ = fmt.Fprintf(stdout, "registries: %d\n", len(cfg.Registries))
	for _, status := range statuses {
		_, _ = fmt.Fprintf(stdout, "registry %s: %s %s skills=%d generated_at=%s\n", status.Name, status.Type, status.Location, status.SkillCount, status.GeneratedAt)
	}
	_, _ = fmt.Fprintf(stdout, "installed: %d\n", len(lock.Skills))
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

func withCLIHint(err error) error {
	if err == nil {
		return nil
	}
	message := err.Error()
	switch {
	case strings.Contains(message, "unknown registry "):
		return fmt.Errorf("%w\nhint: run `skillhub registry list` to see configured registries", err)
	case strings.Contains(message, "target already exists"):
		return fmt.Errorf("%w\nhint: use --force to overwrite the runtime copy, or remove the target directory first", err)
	case strings.Contains(message, "no rollback history for "):
		return fmt.Errorf("%w\nhint: install a newer version or reinstall the Skill once before rolling back", err)
	default:
		return err
	}
}

func usage(stderr io.Writer) error {
	_, _ = fmt.Fprintln(stderr, rootUsage)
	_, _ = fmt.Fprintln(stderr, "run `skillhub help` for available commands")
	return fmt.Errorf("invalid command")
}
