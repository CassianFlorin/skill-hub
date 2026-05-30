package tui

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/cassian/skill-hub/internal/manage"
	tea "github.com/charmbracelet/bubbletea"
)

func TestInitialModelShowsLocalSkills(t *testing.T) {
	model := NewModel(manage.SummaryView{
		Skills: []manage.SkillView{
			{
				Scope:    manage.ScopeGlobal,
				Identity: "official/git-commit-cn",
				Version:  "0.1.0",
				RuntimeStates: map[string]string{
					"codex": "missing",
				},
			},
			{
				Scope:    manage.ScopeProject,
				Identity: "project/commerce-data-fix-sql",
				Version:  "unversioned",
				Path:     ".codex/skills/commerce-data-fix-sql",
			},
		},
	}, nil)

	view := model.View()

	assertViewContains(t, view, "Local")
	assertViewContains(t, view, "official/git-commit-cn")
	assertViewContains(t, view, "global")
	assertViewContains(t, view, "project/commerce-data-fix-sql")
	assertViewContains(t, view, "project")
}

func TestModelTabsShowCatalogAndLogs(t *testing.T) {
	model := NewModel(manage.SummaryView{}, []manage.CatalogSkill{
		{
			Registry: "hub",
			Identity: "official/git-commit-cn",
			Version:  "0.1.0",
			Trust:    "official",
		},
	})

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = next.(Model)
	catalogView := model.View()
	assertViewContains(t, catalogView, "Catalog")
	assertViewContains(t, catalogView, "official/git-commit-cn")
	assertViewContains(t, catalogView, "hub")

	next, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = next.(Model)
	deploymentView := model.View()
	assertViewContains(t, deploymentView, "Deployments")

	next, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = next.(Model)
	logView := model.View()
	assertViewContains(t, logView, "Logs")
	assertViewContains(t, logView, "No operations yet")
}

func TestModelRunsDirectCatalogInstallAndLogsResult(t *testing.T) {
	var requests []manage.OperationRequest
	model := NewModel(manage.SummaryView{}, []manage.CatalogSkill{
		{Registry: "hub", Identity: "official/git-commit-cn", Version: "0.1.0"},
	}).WithRunner(func(request manage.OperationRequest) (manage.OperationResult, error) {
		requests = append(requests, request)
		return manage.OperationResult{Command: "skillhub install hub/official/git-commit-cn", Message: "installed official/git-commit-cn@0.1.0"}, nil
	})

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = next.(Model)
	next, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = next.(Model)
	next, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = next.(Model)
	next, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = next.(Model)

	assertEqual(t, len(requests), 1, "request count")
	assertEqual(t, requests[0].Kind, manage.OperationInstall, "operation kind")
	assertEqual(t, requests[0].Spec, "hub/official/git-commit-cn", "install spec")
	assertViewContains(t, model.View(), "skillhub install hub/official/git-commit-cn")
	assertViewContains(t, model.View(), "installed official/git-commit-cn@0.1.0")
}

func TestModelRequiresConfirmationForUninstall(t *testing.T) {
	var requests []manage.OperationRequest
	model := NewModel(manage.SummaryView{
		Skills: []manage.SkillView{{Scope: manage.ScopeGlobal, Identity: "official/git-commit-cn", Version: "0.1.0"}},
	}, nil).WithRunner(func(request manage.OperationRequest) (manage.OperationResult, error) {
		requests = append(requests, request)
		return manage.OperationResult{Command: "skillhub uninstall official/git-commit-cn", Message: "uninstalled official/git-commit-cn"}, nil
	})

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	model = next.(Model)

	assertEqual(t, len(requests), 0, "request count before confirm")
	assertViewContains(t, model.View(), "Confirm uninstall official/git-commit-cn")

	next, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	model = next.(Model)
	next, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = next.(Model)
	next, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = next.(Model)
	next, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = next.(Model)

	assertEqual(t, len(requests), 1, "request count after confirm")
	assertEqual(t, requests[0].Kind, manage.OperationUninstall, "operation kind")
	assertEqual(t, requests[0].Identity, "official/git-commit-cn", "identity")
	assertViewContains(t, model.View(), "uninstalled official/git-commit-cn")
}

func assertViewContains(t *testing.T, view string, want string) {
	t.Helper()
	if !strings.Contains(view, want) {
		t.Fatalf("view missing %q:\n%s", want, view)
	}
}

func assertEqual[T comparable](t *testing.T, got T, want T, label string) {
	t.Helper()
	if got != want {
		gotJSON, _ := json.Marshal(got)
		wantJSON, _ := json.Marshal(want)
		t.Fatalf("%s: got %s, want %s", label, gotJSON, wantJSON)
	}
}
