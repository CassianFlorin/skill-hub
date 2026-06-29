package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/cassian/skill-hub/internal/install"
	"github.com/cassian/skill-hub/internal/registry"
)

func runCheck(args []string, stdout io.Writer) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: skillhub check")
	}
	plans, err := install.PlanUpdateDetails(install.UpdateOptions{})
	if err != nil {
		return err
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
	identity := ""
	for _, arg := range args {
		switch arg {
		case "--preview", "--dry-run":
			preview = true
		default:
			if strings.HasPrefix(arg, "--") || identity != "" {
				return fmt.Errorf("usage: skillhub update [identity] [--preview|--dry-run]")
			}
			identity = arg
		}
	}
	if preview {
		plans, err := install.PlanUpdateDetails(install.UpdateOptions{Identity: identity})
		if err != nil {
			return err
		}
		return printUpdatePreview(stdout, plans)
	}
	var changes [][3]string
	var err error
	plans, planErr := install.PlanUpdateDetails(install.UpdateOptions{Identity: identity})
	if identity != "" {
		changes, err = install.UpdateOne(identity)
	} else {
		changes, err = install.UpdateAll()
	}
	if err != nil {
		return err
	}
	heldSkipped := heldUpdatePlans(plans, planErr)
	if len(changes) == 0 && len(heldSkipped) == 0 {
		_, _ = fmt.Fprintln(stdout, "all skills are current")
		return nil
	}
	for _, plan := range heldSkipped {
		_, _ = fmt.Fprintf(stdout, "skipped held %s\n", plan.Identity)
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
		if plan.Held {
			policy = "held"
		}
		rows = append(rows, []string{plan.Identity, plan.CurrentVersion, plan.AvailableVersion, policy, updatePlanSource(plan)})
	}
	return formatTable([]string{"Skill", "Current", "Available", "Policy", "Source"}, rows)
}

func printUpdatePreview(stdout io.Writer, plans []install.UpdatePlan) error {
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
