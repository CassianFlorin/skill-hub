package manage

import (
	"sort"
	"strings"

	"github.com/CassianFlorin/skill-hub/internal/config"
	"github.com/CassianFlorin/skill-hub/internal/deploy"
	"github.com/CassianFlorin/skill-hub/internal/install"
	projectskills "github.com/CassianFlorin/skill-hub/internal/project"
	"github.com/CassianFlorin/skill-hub/internal/registry"
)

const (
	ScopeGlobal  = "global"
	ScopeProject = "project"
)

type SummaryView struct {
	Skills []SkillView
}

type SkillView struct {
	Scope         string
	Identity      string
	Version       string
	Description   string
	Source        string
	Path          string
	RuntimeStates map[string]string
}

type CatalogSkill struct {
	Registry    string
	Identity    string
	Name        string
	Namespace   string
	Version     string
	Description string
	Targets     []string
	Tags        []string
	Trust       string
	Featured    bool
}

type OperationKind string

const (
	OperationInstall          OperationKind = "install"
	OperationRegistrySync     OperationKind = "registry-sync"
	OperationUpdate           OperationKind = "update"
	OperationDeploy           OperationKind = "deploy"
	OperationUninstall        OperationKind = "uninstall"
	OperationRollback         OperationKind = "rollback"
	OperationRegistryDelete   OperationKind = "registry-delete"
	OperationProjectOverwrite OperationKind = "project-overwrite"
)

type OperationRequest struct {
	Kind     OperationKind
	Spec     string
	Identity string
	Runtime  string
	Force    bool
}

type OperationResult struct {
	Command string
	Message string
}

func Summary(workDir string) (SummaryView, error) {
	lock, err := install.LoadLock()
	if err != nil {
		return SummaryView{}, err
	}
	statusMap, err := deploymentStatusMap()
	if err != nil {
		return SummaryView{}, err
	}
	var skills []SkillView
	for _, locked := range lock.Skills {
		identity := locked.DisplayIdentity()
		skills = append(skills, SkillView{
			Scope:         ScopeGlobal,
			Identity:      identity,
			Version:       locked.Version,
			Description:   locked.Description,
			Source:        locked.SourceType,
			Path:          locked.InstalledPath,
			RuntimeStates: statusMap[identity],
		})
	}
	projectSkills, err := projectskills.DiscoverSkills(workDir)
	if err != nil {
		return SummaryView{}, err
	}
	for _, found := range projectSkills {
		skills = append(skills, SkillView{
			Scope:         ScopeProject,
			Identity:      found.Identity,
			Version:       found.Version,
			Path:          found.RelPath,
			RuntimeStates: map[string]string{},
		})
	}
	sort.Slice(skills, func(i, j int) bool {
		if skills[i].Scope != skills[j].Scope {
			return skills[i].Scope < skills[j].Scope
		}
		return skills[i].Identity < skills[j].Identity
	})
	return SummaryView{Skills: skills}, nil
}

func SearchCatalog(workDir string, query string) ([]CatalogSkill, error) {
	cfg, err := config.Load(workDir)
	if err != nil {
		return nil, err
	}
	results, err := registry.SearchIndexes(cfg, query)
	if err != nil {
		return nil, err
	}
	catalog := make([]CatalogSkill, 0, len(results))
	for _, result := range results {
		indexed := result.Skill
		catalog = append(catalog, CatalogSkill{
			Registry:    result.Registry,
			Identity:    indexed.Identity,
			Name:        indexed.Name,
			Namespace:   indexed.Namespace,
			Version:     indexed.Version,
			Description: indexed.Description,
			Targets:     indexed.Targets,
			Tags:        indexed.Tags,
			Trust:       indexed.Trust.Level,
			Featured:    indexed.Featured,
		})
	}
	return catalog, nil
}

func RequiresConfirmation(request OperationRequest) bool {
	switch request.Kind {
	case OperationUninstall, OperationRollback, OperationRegistryDelete, OperationProjectOverwrite:
		return true
	case OperationDeploy:
		return request.Force
	default:
		return false
	}
}

func Execute(workDir string, request OperationRequest) (OperationResult, error) {
	switch request.Kind {
	case OperationInstall:
		locked, err := install.Install(workDir, request.Spec)
		if err != nil {
			return OperationResult{}, err
		}
		return OperationResult{
			Command: "skillhub install " + request.Spec,
			Message: "installed " + locked.DisplayIdentity() + "@" + locked.Version,
		}, nil
	case OperationUpdate:
		changes, skipped, err := install.Update(install.UpdateOptions{})
		if err != nil {
			return OperationResult{}, err
		}
		return OperationResult{
			Command: "skillhub update",
			Message: updateMessage(changes, skipped),
		}, nil
	case OperationDeploy:
		runtime := request.Runtime
		if runtime == "" {
			runtime = "codex"
		}
		results, err := deployRuntime(runtime, deploy.Options{Identity: request.Identity, Force: request.Force})
		if err != nil {
			return OperationResult{}, err
		}
		return OperationResult{
			Command: deployCommand(runtime, request),
			Message: deployMessage(results),
		}, nil
	case OperationRollback:
		locked, err := install.Rollback(request.Identity)
		if err != nil {
			return OperationResult{}, err
		}
		return OperationResult{
			Command: "skillhub rollback " + request.Identity,
			Message: "rolled back " + locked.DisplayIdentity() + "@" + locked.Version,
		}, nil
	case OperationUninstall:
		locked, err := install.Uninstall(request.Identity)
		if err != nil {
			return OperationResult{}, err
		}
		return OperationResult{
			Command: "skillhub uninstall " + request.Identity,
			Message: "uninstalled " + locked.DisplayIdentity(),
		}, nil
	default:
		return OperationResult{}, nil
	}
}

func deployRuntime(runtime string, options deploy.Options) ([]deploy.Result, error) {
	return deploy.DeployRuntime(runtime, options)
}

func deployUnsupported(runtime string) error {
	_, err := deploy.RuntimeDir(runtime)
	return err
}

func deployCommand(runtime string, request OperationRequest) string {
	command := "skillhub deploy " + runtime
	if request.Identity != "" {
		command += " " + request.Identity
	}
	if request.Force {
		command += " --force"
	}
	return command
}

func deployMessage(results []deploy.Result) string {
	if len(results) == 0 {
		return "no Skills deployed"
	}
	var parts []string
	for _, result := range results {
		parts = append(parts, result.Identity+" "+result.State)
	}
	return strings.Join(parts, "; ")
}

func updateMessage(changes [][3]string, skipped []install.SkippedUpdate) string {
	if len(changes) == 0 && len(skipped) == 0 {
		return "all installed Skills already up to date"
	}
	var parts []string
	for _, change := range changes {
		parts = append(parts, change[0]+" "+change[1]+" -> "+change[2])
	}
	for _, skip := range skipped {
		parts = append(parts, "skipped "+skip.Identity+" "+skip.CurrentVersion+" -> "+skip.AvailableVersion+" ("+skip.Reason+"; run skillhub update "+skip.Identity+" --major)")
	}
	if len(changes) > 0 {
		parts = append(parts, "runtime copies were not changed")
	}
	return strings.Join(parts, "; ")
}

func deploymentStatusMap() (map[string]map[string]string, error) {
	statuses, err := deploy.Statuses("")
	if err != nil {
		return nil, err
	}
	result := map[string]map[string]string{}
	for _, status := range statuses {
		if result[status.Identity] == nil {
			result[status.Identity] = map[string]string{}
		}
		result[status.Identity][status.Runtime] = status.State
	}
	return result, nil
}
