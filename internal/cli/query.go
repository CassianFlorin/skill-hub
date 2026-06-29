package cli

import (
	"fmt"
	"io"

	"github.com/cassian/skill-hub/internal/config"
	"github.com/cassian/skill-hub/internal/registry"
)

func runSearch(args []string, stdout io.Writer, workDir string) error {
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("usage: skillhub search <query> [--json]")
	}
	query := args[0]
	jsonOutput := false
	if len(args) == 2 {
		if args[1] != "--json" {
			return fmt.Errorf("usage: skillhub search <query> [--json]")
		}
		jsonOutput = true
	}
	cfg, err := config.Load(workDir)
	if err != nil {
		return err
	}
	results, err := registry.SearchIndexes(cfg, query)
	if err != nil {
		return err
	}
	if jsonOutput {
		return writeJSON(stdout, catalogJSONResults(results))
	}
	if len(results) == 0 {
		_, _ = fmt.Fprintln(stdout, "no skills found")
		return nil
	}
	_, _ = fmt.Fprint(stdout, formatCatalogResults(results))
	return nil
}

func runInfo(args []string, stdout io.Writer, workDir string) error {
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("usage: skillhub info <registry/identity|identity> [--json]")
	}
	spec := args[0]
	jsonOutput := false
	if len(args) == 2 {
		if args[1] != "--json" {
			return fmt.Errorf("usage: skillhub info <registry/identity|identity> [--json]")
		}
		jsonOutput = true
	}
	cfg, err := config.Load(workDir)
	if err != nil {
		return err
	}
	result, ok, err := registry.FindIndexedSkill(cfg, spec)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("skill %q not found", spec)
	}
	indexed := result.Skill
	installCommand := fmt.Sprintf("skillhub install %s/%s", result.Registry, indexed.Identity)
	if jsonOutput {
		return writeJSON(stdout, infoJSONResult{
			Registry:       result.Registry,
			Skill:          indexed,
			InstallCommand: installCommand,
		})
	}
	_, _ = fmt.Fprint(stdout, formatInfoResult(result, installCommand))
	return nil
}
