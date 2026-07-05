package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/CassianFlorin/skill-hub/internal/publish"
	"github.com/CassianFlorin/skill-hub/internal/registry"
)

const publishUsage = "usage: skillhub publish <skill-path> --registry <name> [--dest <path>] [--trust <level>] [--branch <name>] [--message <text>] [--dry-run] [--json]"

type publishJSON struct {
	Identity  string               `json:"identity"`
	Version   string               `json:"version"`
	Action    string               `json:"action"`
	Registry  string               `json:"registry"`
	Dest      string               `json:"dest"`
	Checksum  string               `json:"checksum"`
	DryRun    bool                 `json:"dry_run"`
	Pushed    bool                 `json:"pushed"`
	Branch    string               `json:"branch,omitempty"`
	CommitMsg string               `json:"commit_message,omitempty"`
	OldEntry  *registry.IndexSkill `json:"old_entry,omitempty"`
	NewEntry  registry.IndexSkill  `json:"new_entry"`
}

func runPublish(args []string, stdout io.Writer, workDir string) error {
	var skillPath string
	opts := publish.Options{}
	jsonOutput := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--registry", "--dest", "--trust", "--branch", "--message":
			i++
			if i >= len(args) {
				return fmt.Errorf("%s requires a value\n%s", arg, publishUsage)
			}
			value := args[i]
			switch arg {
			case "--registry":
				opts.Registry = value
			case "--dest":
				opts.Dest = value
			case "--trust":
				opts.Trust = value
			case "--branch":
				opts.Branch = value
			case "--message":
				opts.Message = value
			}
		case "--dry-run":
			opts.DryRun = true
		case "--json":
			jsonOutput = true
		default:
			if skillPath != "" {
				return fmt.Errorf(publishUsage)
			}
			skillPath = arg
		}
	}
	if skillPath == "" || opts.Registry == "" {
		return fmt.Errorf(publishUsage)
	}
	result, err := publish.Publish(workDir, skillPath, opts)
	if err != nil {
		return withCLIHint(err)
	}
	if jsonOutput {
		return writeJSON(stdout, publishJSON{
			Identity:  result.Identity,
			Version:   result.Version,
			Action:    result.Action,
			Registry:  result.Registry,
			Dest:      result.Dest,
			Checksum:  result.Checksum,
			DryRun:    result.DryRun,
			Pushed:    result.Pushed,
			Branch:    result.Branch,
			CommitMsg: result.CommitMsg,
			OldEntry:  result.OldEntry,
			NewEntry:  result.NewEntry,
		})
	}
	if result.Action == publish.ActionUnchanged {
		_, _ = fmt.Fprintf(stdout, "%s@%s already published to %s; nothing to do\n", result.Identity, result.Version, result.Registry)
		return nil
	}
	if result.DryRun {
		_, _ = fmt.Fprintf(stdout, "dry-run: would publish %s@%s to %s at %s (%s)\n", result.Identity, result.Version, result.Registry, result.Dest, result.Action)
		printEntryDiff(stdout, result.OldEntry, result.NewEntry)
		return nil
	}
	_, _ = fmt.Fprintf(stdout, "published %s@%s to %s at %s (%s)\n", result.Identity, result.Version, result.Registry, result.Dest, result.Action)
	if result.Pushed {
		branch := result.Branch
		if branch == "" {
			branch = "default branch"
		}
		_, _ = fmt.Fprintf(stdout, "pushed commit %q to %s\n", result.CommitMsg, branch)
	}
	return nil
}

func printEntryDiff(stdout io.Writer, oldEntry *registry.IndexSkill, newEntry registry.IndexSkill) {
	if oldEntry != nil {
		_, _ = fmt.Fprintln(stdout, "current index entry:")
		printEntryJSON(stdout, *oldEntry)
	} else {
		_, _ = fmt.Fprintln(stdout, "current index entry: (none)")
	}
	_, _ = fmt.Fprintln(stdout, "new index entry:")
	printEntryJSON(stdout, newEntry)
}

func printEntryJSON(stdout io.Writer, entry registry.IndexSkill) {
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return
	}
	_, _ = fmt.Fprintln(stdout, string(data))
}
