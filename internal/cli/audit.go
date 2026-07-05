package cli

import (
	"fmt"
	"io"
	"strconv"

	"github.com/CassianFlorin/skill-hub/internal/audit"
)

const auditUsage = "usage: skillhub audit [--limit <n>] [--json]"

func runAudit(args []string, stdout io.Writer) error {
	jsonOutput := false
	limit := 0
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonOutput = true
		case "--limit":
			i++
			if i >= len(args) {
				return fmt.Errorf(auditUsage)
			}
			value, err := strconv.Atoi(args[i])
			if err != nil || value < 1 {
				return fmt.Errorf("--limit requires a positive number\n%s", auditUsage)
			}
			limit = value
		default:
			return fmt.Errorf(auditUsage)
		}
	}
	events, err := audit.List(limit)
	if err != nil {
		return err
	}
	if jsonOutput {
		if events == nil {
			events = []audit.Event{}
		}
		return writeJSON(stdout, events)
	}
	if len(events) == 0 {
		_, _ = fmt.Fprintln(stdout, "no audit events recorded")
		return nil
	}
	rows := make([][]string, 0, len(events))
	for _, event := range events {
		version := event.Version
		if event.FromVersion != "" {
			version = event.FromVersion + " -> " + event.Version
		}
		detail := event.Detail
		if event.Runtime != "" {
			if detail != "" {
				detail = event.Runtime + " " + detail
			} else {
				detail = event.Runtime
			}
		}
		rows = append(rows, []string{event.Time, event.Command, event.Identity, version, event.Result, detail})
	}
	_, _ = fmt.Fprint(stdout, formatTable([]string{"Time", "Command", "Skill", "Version", "Result", "Detail"}, rows))
	path, err := audit.LogPath()
	if err == nil {
		_, _ = fmt.Fprintf(stdout, "\nfull JSONL log: %s\n", path)
	}
	return nil
}
