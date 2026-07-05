package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/CassianFlorin/skill-hub/internal/install"
	"github.com/CassianFlorin/skill-hub/internal/registry"
)

func runCheck(args []string, stdout io.Writer) error {
	jsonOutput := false
	for _, arg := range args {
		if arg != "--json" {
			return fmt.Errorf("usage: skillhub check [--json]")
		}
		jsonOutput = true
	}
	plans, err := install.PlanUpdateDetails(install.UpdateOptions{})
	if err != nil {
		return err
	}
	if jsonOutput {
		return writeJSON(stdout, updatePlanJSONList(plans))
	}
	if len(plans) == 0 {
		_, _ = fmt.Fprintln(stdout, "all skills are current")
		return nil
	}
	_, _ = fmt.Fprintln(stdout, "Updates available:")
	_, _ = fmt.Fprintln(stdout)
	_, _ = fmt.Fprint(stdout, formatUpdatePlanTable(plans))
	_, _ = fmt.Fprintln(stdout)
	_, _ = fmt.Fprintln(stdout, "Run `skillhub update --preview` to inspect changes.")
	_, _ = fmt.Fprintln(stdout, "Run `skillhub update <skill>` to update one Skill.")
	return nil
}

func runUpdate(args []string, stdout io.Writer) error {
	preview := false
	allowMajor := false
	identity := ""
	for _, arg := range args {
		switch arg {
		case "--preview", "--dry-run":
			preview = true
		case "--major":
			allowMajor = true
		default:
			if strings.HasPrefix(arg, "--") || identity != "" {
				return fmt.Errorf("usage: skillhub update [identity] [--major] [--preview|--dry-run]")
			}
			identity = arg
		}
	}
	if preview {
		plans, err := install.PlanUpdateDetails(install.UpdateOptions{Identity: identity})
		if err != nil {
			return err
		}
		return printUpdatePreview(stdout, plans, allowMajor)
	}
	plans, planErr := install.PlanUpdateDetails(install.UpdateOptions{Identity: identity})
	changes, skipped, err := install.Update(install.UpdateOptions{Identity: identity, AllowMajor: allowMajor})
	if err != nil {
		return err
	}
	heldSkipped := heldUpdatePlans(plans, planErr)
	if len(changes) == 0 && len(heldSkipped) == 0 && len(skipped) == 0 {
		_, _ = fmt.Fprintln(stdout, "all skills are current")
		return nil
	}
	for _, plan := range heldSkipped {
		_, _ = fmt.Fprintf(stdout, "skipped held %s\n", plan.Identity)
	}
	for _, skip := range skipped {
		_, _ = fmt.Fprintf(stdout, "skipped %s %s -> %s (%s)\n", skip.Identity, skip.CurrentVersion, skip.AvailableVersion, skip.Reason)
		_, _ = fmt.Fprintf(stdout, "apply it explicitly: skillhub update %s --major\n", skip.Identity)
	}
	for _, change := range changes {
		_, _ = fmt.Fprintf(stdout, "updated %s %s -> %s\n", change[0], change[1], change[2])
		_, _ = fmt.Fprintf(stdout, "rollback: skillhub rollback %s\n", change[0])
	}
	_, _ = fmt.Fprintln(stdout, "runtime copies were not changed; run skillhub deploy status to inspect them")
	return nil
}

func heldUpdatePlans(plans []install.UpdatePlan, err error) []install.UpdatePlan {
	if err != nil {
		return nil
	}
	var held []install.UpdatePlan
	for _, plan := range plans {
		if plan.Held {
			held = append(held, plan)
		}
	}
	return held
}

func formatUpdatePlanTable(plans []install.UpdatePlan) string {
	rows := make([][]string, 0, len(plans))
	for _, plan := range plans {
		policy := "update"
		switch {
		case plan.Held:
			policy = "held"
		case plan.Breaking:
			policy = "breaking (--major)"
		case plan.Major:
			policy = "major (--major)"
		}
		rows = append(rows, []string{plan.Identity, plan.CurrentVersion, plan.AvailableVersion, policy, updatePlanSource(plan)})
	}
	return formatTable([]string{"Skill", "Current", "Available", "Policy", "Source"}, rows)
}

func printUpdatePreview(stdout io.Writer, plans []install.UpdatePlan, allowMajor bool) error {
	if len(plans) == 0 {
		_, _ = fmt.Fprintln(stdout, "all skills are current")
		return nil
	}
	_, _ = fmt.Fprintln(stdout, "Update preview:")
	for _, plan := range plans {
		_, _ = fmt.Fprintf(stdout, "\n%s %s -> %s\n", plan.Identity, plan.CurrentVersion, plan.AvailableVersion)
		_, _ = fmt.Fprintf(stdout, "would update %s %s -> %s\n", plan.Identity, plan.CurrentVersion, plan.AvailableVersion)
		_, _ = fmt.Fprintf(stdout, "  Source: %s\n", updatePlanSource(plan))
		if plan.CurrentCommit != "" || plan.AvailableCommit != "" {
			_, _ = fmt.Fprintf(stdout, "  Commit: %s -> %s\n", shortCommit(plan.CurrentCommit), shortCommit(plan.AvailableCommit))
		}
		if len(plan.Targets) > 0 {
			_, _ = fmt.Fprintf(stdout, "  Targets: %s\n", strings.Join(plan.Targets, ", "))
		}
		if len(plan.DeployedTo) > 0 {
			_, _ = fmt.Fprintf(stdout, "  Deployed runtimes: %s\n", strings.Join(plan.DeployedTo, ", "))
		}
		if plan.Held {
			if plan.HoldReason != "" {
				_, _ = fmt.Fprintf(stdout, "  Held: %s\n", plan.HoldReason)
			} else {
				_, _ = fmt.Fprintln(stdout, "  Held: yes")
			}
			_, _ = fmt.Fprintln(stdout, "  Skipped: held")
		}
		if !plan.Held && !allowMajor {
			switch {
			case plan.Breaking:
				_, _ = fmt.Fprintf(stdout, "  Skipped: breaking change; apply with skillhub update %s --major\n", plan.Identity)
			case plan.Major:
				_, _ = fmt.Fprintf(stdout, "  Skipped: major update; apply with skillhub update %s --major\n", plan.Identity)
			}
		}
		_, _ = fmt.Fprintf(stdout, "  Update one Skill: skillhub update %s\n", plan.Identity)
		_, _ = fmt.Fprintf(stdout, "  Roll back after update: skillhub rollback %s\n", plan.Identity)
	}
	_, _ = fmt.Fprintln(stdout, "\nupdates managed store only; runtime copies will not be changed")
	return nil
}

func updatePlanSource(plan install.UpdatePlan) string {
	switch plan.SourceType {
	case registry.SourceTypeGit:
		if plan.SourceRegistry != "" {
			return "git " + plan.SourceRegistry
		}
		return "git " + plan.SourceURL
	case registry.SourceTypeRegistry:
		if plan.SourceRegistry != "" {
			return "local " + plan.SourceRegistry
		}
		return "local registry"
	}
	if plan.SourceType != "" {
		return plan.SourceType
	}
	return "unknown"
}

func shortCommit(commit string) string {
	if len(commit) > 12 {
		return commit[:12]
	}
	if commit == "" {
		return "unknown"
	}
	return commit
}
