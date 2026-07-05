package cli

import (
	"fmt"
	"io"

	"github.com/CassianFlorin/skill-hub/internal/install"
	projectskills "github.com/CassianFlorin/skill-hub/internal/project"
)

func runList(args []string, stdout io.Writer, workDir string) error {
	scope := "all"
	jsonOutput := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--scope":
			i++
			if i >= len(args) {
				return fmt.Errorf("--scope requires a value")
			}
			scope = args[i]
		case "--json":
			jsonOutput = true
		default:
			return fmt.Errorf(listUsage)
		}
	}
	if scope != "all" && scope != "global" && scope != "project" {
		return fmt.Errorf("unsupported scope %q", scope)
	}

	lock, err := install.LoadLock()
	if err != nil {
		return err
	}
	projectSkills, err := projectskills.DiscoverSkills(workDir)
	if err != nil {
		return err
	}

	var rows []skillListRow
	if scope == "all" || scope == "global" {
		for _, locked := range lock.Skills {
			rows = append(rows, skillListRow{
				Scope:    "global",
				Skill:    locked.DisplayIdentity(),
				Version:  locked.Version,
				Location: locked.Description,
			})
		}
	}
	if scope == "all" || scope == "project" {
		for _, discovered := range projectSkills {
			rows = append(rows, skillListRow{
				Scope:    "project",
				Skill:    discovered.Identity,
				Version:  discovered.Version,
				Location: discovered.RelPath,
			})
		}
	}
	if jsonOutput {
		return writeJSON(stdout, listJSONRows(rows))
	}
	if len(rows) == 0 {
		_, _ = fmt.Fprintln(stdout, "no skills found")
		return nil
	}
	_, _ = fmt.Fprint(stdout, formatSkillList(rows))
	return nil
}

type skillListRow struct {
	Scope    string
	Skill    string
	Version  string
	Location string
}

func formatSkillList(rows []skillListRow) string {
	tableRows := make([][]string, 0, len(rows))
	for _, row := range rows {
		tableRows = append(tableRows, []string{row.Scope, row.Skill, row.Version, row.Location})
	}
	return formatTable([]string{"Scope", "Skill", "Version", "Location / Description"}, tableRows)
}
