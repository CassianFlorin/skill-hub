package cli

import (
	"github.com/cassian/skill-hub/internal/deploy"
	"github.com/cassian/skill-hub/internal/install"
	"github.com/cassian/skill-hub/internal/registry"
)

// listJSONRow is the machine-readable form of a `skillhub list` row.
type listJSONRow struct {
	Scope    string `json:"scope"`
	Skill    string `json:"skill"`
	Version  string `json:"version"`
	Location string `json:"location"`
}

func listJSONRows(rows []skillListRow) []listJSONRow {
	out := make([]listJSONRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, listJSONRow{Scope: r.Scope, Skill: r.Skill, Version: r.Version, Location: r.Location})
	}
	return out
}

// registryJSON is the machine-readable form of a configured registry status.
type registryJSON struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Location    string `json:"location"`
	SkillCount  int    `json:"skill_count"`
	GeneratedAt string `json:"generated_at"`
}

func registryJSONList(statuses []registry.RegistryStatus) []registryJSON {
	out := make([]registryJSON, 0, len(statuses))
	for _, s := range statuses {
		out = append(out, registryJSON{
			Name:        s.Name,
			Type:        s.Type,
			Location:    s.Location,
			SkillCount:  s.SkillCount,
			GeneratedAt: s.GeneratedAt,
		})
	}
	return out
}

// doctorRuntimeJSON pairs a runtime name with its resolved directory.
type doctorRuntimeJSON struct {
	Name string `json:"name"`
	Dir  string `json:"dir"`
}

// doctorJSON is the machine-readable form of `skillhub doctor`.
type doctorJSON struct {
	Config     string              `json:"config"`
	ConfigPath string              `json:"config_path"`
	Home       string              `json:"home"`
	InstallDir string              `json:"install_dir"`
	Runtimes   []doctorRuntimeJSON `json:"runtimes"`
	Registries []registryJSON      `json:"registries"`
	Installed  int                 `json:"installed"`
}

// deployStatusJSON is the machine-readable form of a `skillhub deploy status` row.
type deployStatusJSON struct {
	Identity string `json:"identity"`
	Runtime  string `json:"runtime"`
	State    string `json:"state"`
}

func deployStatusJSONList(statuses []deploy.Status) []deployStatusJSON {
	out := make([]deployStatusJSON, 0, len(statuses))
	for _, s := range statuses {
		out = append(out, deployStatusJSON{Identity: s.Identity, Runtime: s.Runtime, State: s.State})
	}
	return out
}

// updatePlanJSON is the machine-readable form of an available update.
type updatePlanJSON struct {
	Identity         string   `json:"identity"`
	CurrentVersion   string   `json:"current_version"`
	AvailableVersion string   `json:"available_version"`
	CurrentCommit    string   `json:"current_commit,omitempty"`
	AvailableCommit  string   `json:"available_commit,omitempty"`
	Source           string   `json:"source"`
	Targets          []string `json:"targets,omitempty"`
	DeployedTo       []string `json:"deployed_to,omitempty"`
	Held             bool     `json:"held"`
	HoldReason       string   `json:"hold_reason,omitempty"`
}

func updatePlanJSONList(plans []install.UpdatePlan) []updatePlanJSON {
	out := make([]updatePlanJSON, 0, len(plans))
	for _, p := range plans {
		out = append(out, updatePlanJSON{
			Identity:         p.Identity,
			CurrentVersion:   p.CurrentVersion,
			AvailableVersion: p.AvailableVersion,
			CurrentCommit:    p.CurrentCommit,
			AvailableCommit:  p.AvailableCommit,
			Source:           updatePlanSource(p),
			Targets:          p.Targets,
			DeployedTo:       p.DeployedTo,
			Held:             p.Held,
			HoldReason:       p.HoldReason,
		})
	}
	return out
}

// holdJSON is the machine-readable form of a held Skill.
type holdJSON struct {
	Skill   string `json:"skill"`
	Version string `json:"version"`
	Reason  string `json:"reason,omitempty"`
	HeldAt  string `json:"held_at"`
}

func holdJSONList(held []install.LockedSkill) []holdJSON {
	out := make([]holdJSON, 0, len(held))
	for _, locked := range held {
		reason := ""
		created := ""
		if locked.Hold != nil {
			reason = locked.Hold.Reason
			created = locked.Hold.CreatedAt
		}
		out = append(out, holdJSON{Skill: locked.DisplayIdentity(), Version: locked.Version, Reason: reason, HeldAt: created})
	}
	return out
}

// historyJSON is the machine-readable form of a rollback history entry.
type historyJSON struct {
	Identity     string `json:"identity"`
	Version      string `json:"version"`
	SourceRef    string `json:"source_ref,omitempty"`
	SourceCommit string `json:"source_commit,omitempty"`
	CreatedAt    string `json:"created_at"`
}

func historyJSONList(entries []install.HistoryEntry) []historyJSON {
	out := make([]historyJSON, 0, len(entries))
	for _, e := range entries {
		out = append(out, historyJSON{
			Identity:     e.Identity,
			Version:      e.Version,
			SourceRef:    e.SourceRef,
			SourceCommit: e.SourceCommit,
			CreatedAt:    e.CreatedAt,
		})
	}
	return out
}
