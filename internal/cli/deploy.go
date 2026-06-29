package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/cassian/skill-hub/internal/deploy"
)

func deployUsage() string {
	return fmt.Sprintf("usage: skillhub deploy <%s> [identity] [--dry-run] [--force] [--profile <name>]", strings.Join(deploy.RuntimeNames(), "|"))
}

func supportedRuntimeList() string {
	return strings.Join(deploy.RuntimeNames(), ", ")
}

func runDeploy(args []string, stdout io.Writer) error {
	if len(args) < 1 {
		return errors.New(deployUsage())
	}
	runtime := args[0]
	if runtime == "status" {
		return runDeployStatus(args[1:], stdout)
	}
	options := deploy.Options{}
	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--dry-run":
			options.DryRun = true
		case "--force":
			options.Force = true
		case "--profile":
			i++
			if i >= len(args) {
				return fmt.Errorf("--profile requires a value")
			}
			options.Profile = args[i]
		default:
			if options.Identity != "" {
				return errors.New(deployUsage())
			}
			options.Identity = arg
		}
	}
	deployed, err := deploy.DeployRuntime(runtime, options)
	if err != nil {
		if strings.Contains(err.Error(), "unsupported runtime ") {
			return fmt.Errorf("%w; supported runtimes: %s", err, supportedRuntimeList())
		}
		return withCLIHint(err)
	}
	for _, result := range deployed {
		_, _ = fmt.Fprintln(stdout, deployResultLine(result, runtime))
	}
	return nil
}

func deployResultLine(result deploy.Result, runtime string) string {
	if result.Runtime != "" {
		runtime = result.Runtime
	}
	switch result.State {
	case deploy.StateSkipped:
		return fmt.Sprintf("skipped %s to %s: %s", result.Identity, runtime, result.Reason)
	case deploy.StateConflict:
		return fmt.Sprintf("conflict %s to %s: %s", result.Identity, runtime, result.Reason)
	case deploy.StateWouldDeploy:
		return fmt.Sprintf("would deploy %s to %s", result.Identity, runtime)
	default:
		if result.DryRun {
			return fmt.Sprintf("would deploy %s to %s", result.Identity, runtime)
		}
		return fmt.Sprintf("deployed %s to %s", result.Identity, runtime)
	}
}

func runDeployStatus(args []string, stdout io.Writer) error {
	if len(args) > 1 {
		return fmt.Errorf("usage: skillhub deploy status [%s]", strings.Join(deploy.RuntimeNames(), "|"))
	}
	runtime := ""
	if len(args) == 1 {
		runtime = args[0]
	}
	statuses, err := deploy.Statuses(runtime)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprint(stdout, formatDeployStatus(statuses))
	return nil
}

func formatDeployStatus(statuses []deploy.Status) string {
	rows := make([][]string, 0, len(statuses))
	for _, status := range statuses {
		rows = append(rows, []string{status.Identity, status.Runtime, status.State})
	}
	return formatTable([]string{"Skill", "Runtime", "State"}, rows)
}
