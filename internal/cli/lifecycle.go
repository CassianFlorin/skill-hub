package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cassian/skill-hub/internal/deploy"
	"github.com/cassian/skill-hub/internal/install"
)

func runHold(args []string, stdout io.Writer) error {
	if len(args) != 1 && len(args) != 3 {
		return fmt.Errorf("usage: skillhub hold <identity> [--reason <text>]")
	}
	reason := ""
	if len(args) == 3 {
		if args[1] != "--reason" {
			return fmt.Errorf("usage: skillhub hold <identity> [--reason <text>]")
		}
		reason = args[2]
	}
	locked, err := install.Hold(args[0], reason)
	if err != nil {
		return err
	}
	if reason != "" {
		_, _ = fmt.Fprintf(stdout, "held %s: %s\n", locked.DisplayIdentity(), reason)
		return nil
	}
	_, _ = fmt.Fprintf(stdout, "held %s\n", locked.DisplayIdentity())
	return nil
}

func runUnhold(args []string, stdout io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: skillhub unhold <identity>")
	}
	locked, err := install.Unhold(args[0])
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "unheld %s\n", locked.DisplayIdentity())
	return nil
}

func runHolds(args []string, stdout io.Writer) error {
	jsonOutput := false
	for _, arg := range args {
		if arg != "--json" {
			return fmt.Errorf("usage: skillhub holds [--json]")
		}
		jsonOutput = true
	}
	held, err := install.HeldSkills()
	if err != nil {
		return err
	}
	if jsonOutput {
		return writeJSON(stdout, holdJSONList(held))
	}
	if len(held) == 0 {
		_, _ = fmt.Fprintln(stdout, "no held skills")
		return nil
	}
	rows := make([][]string, 0, len(held))
	for _, locked := range held {
		reason := ""
		created := ""
		if locked.Hold != nil {
			reason = locked.Hold.Reason
			created = locked.Hold.CreatedAt
		}
		rows = append(rows, []string{locked.DisplayIdentity(), locked.Version, reason, created})
	}
	_, _ = fmt.Fprint(stdout, formatTable([]string{"Skill", "Version", "Reason", "Held At"}, rows))
	return nil
}

func runUninstall(args []string, stdout io.Writer) error {
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("usage: skillhub uninstall <identity> [--deployed]")
	}
	removeDeployed := false
	if len(args) == 2 {
		if args[1] != "--deployed" {
			return fmt.Errorf("usage: skillhub uninstall <identity> [--deployed]")
		}
		removeDeployed = true
	}
	locked, err := install.Uninstall(args[0])
	if err != nil {
		return err
	}
	if removeDeployed {
		for _, runtime := range deploy.RuntimeNames() {
			target, err := deploy.RuntimeTarget(runtime, locked.DisplayIdentity())
			if err != nil {
				return err
			}
			if err := os.RemoveAll(target); err != nil {
				return err
			}
		}
	}
	_, _ = fmt.Fprintf(stdout, "uninstalled %s\n", locked.DisplayIdentity())
	return nil
}

func runRollback(args []string, stdout io.Writer) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: skillhub rollback <identity> [--to <version>] [--deploy <runtime>] [--profile <name>]")
	}
	identity := args[0]
	options := install.RollbackOptions{}
	deployRuntime := ""
	deployProfile := ""
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--to":
			i++
			if i >= len(args) {
				return fmt.Errorf("--to requires a version")
			}
			options.To = args[i]
		case "--deploy":
			i++
			if i >= len(args) {
				return fmt.Errorf("--deploy requires a runtime")
			}
			deployRuntime = args[i]
		case "--profile":
			i++
			if i >= len(args) {
				return fmt.Errorf("--profile requires a value")
			}
			deployProfile = args[i]
		default:
			return fmt.Errorf("usage: skillhub rollback <identity> [--to <version>] [--deploy <runtime>] [--profile <name>]")
		}
	}
	if deployProfile != "" && deployRuntime == "" {
		return fmt.Errorf("--profile requires --deploy")
	}
	locked, err := install.RollbackWithOptions(identity, options)
	if err != nil {
		return withCLIHint(err)
	}
	_, _ = fmt.Fprintf(stdout, "rolled back %s to %s\n", locked.DisplayIdentity(), locked.Version)
	if deployRuntime != "" {
		results, err := deploy.DeployRuntime(deployRuntime, deploy.Options{Identity: locked.DisplayIdentity(), Force: true, Profile: deployProfile})
		if err != nil {
			return withCLIHint(err)
		}
		for _, result := range results {
			_, _ = fmt.Fprintln(stdout, deployResultLine(result, deployRuntime))
		}
	}
	return nil
}

func runHistory(args []string, stdout io.Writer) error {
	identity := ""
	jsonOutput := false
	for _, arg := range args {
		switch arg {
		case "--json":
			jsonOutput = true
		default:
			if strings.HasPrefix(arg, "--") || identity != "" {
				return fmt.Errorf("usage: skillhub history <identity> [--json]")
			}
			identity = arg
		}
	}
	if identity == "" {
		return fmt.Errorf("usage: skillhub history <identity> [--json]")
	}
	entries, err := install.History(identity)
	if err != nil {
		return withCLIHint(err)
	}
	if jsonOutput {
		return writeJSON(stdout, historyJSONList(entries))
	}
	if len(entries) == 0 {
		_, _ = fmt.Fprintf(stdout, "No rollback history for %s.\n", identity)
		return nil
	}
	rows := make([][]string, 0, len(entries))
	for _, entry := range entries {
		commit := shortCommit(entry.SourceCommit)
		if entry.SourceCommit == "" {
			commit = "-"
		}
		ref := entry.SourceRef
		if ref == "" {
			ref = "-"
		}
		rows = append(rows, []string{entry.Identity, entry.Version, ref, commit, entry.CreatedAt})
	}
	_, _ = fmt.Fprint(stdout, formatTable([]string{"Skill", "Version", "Ref", "Commit", "Saved"}, rows))
	last := entries[len(entries)-1]
	_, _ = fmt.Fprintf(stdout, "rollback: skillhub rollback %s --to %s\n", last.Identity, last.Version)
	return nil
}
