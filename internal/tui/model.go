package tui

import (
	"fmt"
	"strings"

	"github.com/CassianFlorin/skill-hub/internal/manage"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tab int

const (
	tabLocal tab = iota
	tabCatalog
	tabDeployments
	tabLogs
)

var tabNames = []string{"Local", "Catalog", "Deployments", "Logs"}

type Model struct {
	active   tab
	selected int
	pending  *manage.OperationRequest
	summary  manage.SummaryView
	catalog  []manage.CatalogSkill
	logs     []string
	runner   func(manage.OperationRequest) (manage.OperationResult, error)
}

func NewModel(summary manage.SummaryView, catalog []manage.CatalogSkill) Model {
	return Model{
		active:  tabLocal,
		summary: summary,
		catalog: catalog,
		logs:    []string{},
		runner: func(manage.OperationRequest) (manage.OperationResult, error) {
			return manage.OperationResult{Message: "operation runner is not configured"}, nil
		},
	}
}

func (m Model) WithRunner(runner func(manage.OperationRequest) (manage.OperationResult, error)) Model {
	m.runner = runner
	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			if m.pending != nil {
				m.pending = nil
				m.logs = append(m.logs, "confirmation cancelled")
				return m, nil
			}
			return m, tea.Quit
		case "tab", "right", "l":
			m.active = (m.active + 1) % tab(len(tabNames))
			m.selected = 0
		case "shift+tab", "left", "h":
			m.active = (m.active + tab(len(tabNames)) - 1) % tab(len(tabNames))
			m.selected = 0
		case "down", "j":
			m.moveSelection(1)
		case "up", "k":
			m.moveSelection(-1)
		case "enter":
			m.runSelectedCatalogInstall()
		case "d":
			m.runLocalOperation(manage.OperationDeploy)
		case "u":
			m.runOperation(manage.OperationRequest{Kind: manage.OperationUpdate})
		case "x":
			m.confirmLocalOperation(manage.OperationUninstall)
		case "r":
			m.confirmLocalOperation(manage.OperationRollback)
		case "y":
			if m.pending != nil {
				pending := *m.pending
				m.pending = nil
				m.runOperation(pending)
			}
		case "n":
			if m.pending != nil {
				m.pending = nil
				m.logs = append(m.logs, "confirmation cancelled")
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	var builder strings.Builder
	builder.WriteString(titleStyle.Render("skill-hub TUI"))
	builder.WriteString("\n")
	builder.WriteString(m.renderTabs())
	builder.WriteString("\n\n")
	switch m.active {
	case tabLocal:
		builder.WriteString(m.renderLocal())
	case tabCatalog:
		builder.WriteString(m.renderCatalog())
	case tabDeployments:
		builder.WriteString(m.renderDeployments())
	case tabLogs:
		builder.WriteString(m.renderLogs())
	}
	if confirmation := m.confirmationText(); confirmation != "" {
		builder.WriteString("\n\n")
		builder.WriteString(confirmation)
	}
	builder.WriteString("\n\n")
	builder.WriteString(helpStyle.Render("tab/right/l: next · j/k: select · enter: install · d: deploy · u: update · x: uninstall · r: rollback · q/esc: quit"))
	return builder.String()
}

var (
	titleStyle  = lipgloss.NewStyle().Bold(true)
	activeStyle = lipgloss.NewStyle().Bold(true).Underline(true)
	helpStyle   = lipgloss.NewStyle().Faint(true)
)

func (m Model) renderTabs() string {
	var parts []string
	for i, name := range tabNames {
		if tab(i) == m.active {
			parts = append(parts, activeStyle.Render(name))
			continue
		}
		parts = append(parts, name)
	}
	return strings.Join(parts, "  ")
}

func (m Model) renderLocal() string {
	const boundaryHint = "Managed = skillhub can update and rollback. Project = discovered only. Runtime = what agents load."
	if len(m.summary.Skills) == 0 {
		return "Local\n\n" + boundaryHint + "\n\nNo local Skills found."
	}
	rows := []string{"Local", "", boundaryHint, "", "SCOPE    IDENTITY                         VERSION       RUNTIMES/PATH"}
	for i, skill := range m.summary.Skills {
		right := skill.Path
		if len(skill.RuntimeStates) > 0 {
			right = runtimeSummary(skill.RuntimeStates)
		}
		prefix := " "
		if i == m.selected {
			prefix = ">"
		}
		rows = append(rows, fmt.Sprintf("%s %-8s %-32s %-13s %s", prefix, skill.Scope, skill.Identity, skill.Version, right))
	}
	return strings.Join(rows, "\n")
}

func (m Model) renderCatalog() string {
	if len(m.catalog) == 0 {
		return "Catalog\n\nNo catalog results loaded. Run registry sync first, then search from the CLI or refresh the TUI."
	}
	rows := []string{"Catalog", "", "REGISTRY IDENTITY                         VERSION       TRUST       TAGS"}
	for i, skill := range m.catalog {
		prefix := " "
		if i == m.selected {
			prefix = ">"
		}
		rows = append(rows, fmt.Sprintf("%s %-8s %-32s %-13s %-11s %s", prefix, skill.Registry, skill.Identity, skill.Version, skill.Trust, strings.Join(skill.Tags, ",")))
	}
	return strings.Join(rows, "\n")
}

func (m Model) renderDeployments() string {
	rows := []string{"Deployments", ""}
	if len(m.summary.Skills) == 0 {
		return strings.Join(append(rows, "No installed Skills to deploy."), "\n")
	}
	rows = append(rows, "IDENTITY                         CODEX        CLAUDE       GEMINI")
	for _, skill := range m.summary.Skills {
		if skill.Scope != manage.ScopeGlobal {
			continue
		}
		rows = append(rows, fmt.Sprintf("%-32s %-12s %-12s %s",
			skill.Identity,
			stateOrBlank(skill.RuntimeStates, "codex"),
			stateOrBlank(skill.RuntimeStates, "claude"),
			stateOrBlank(skill.RuntimeStates, "gemini"),
		))
	}
	return strings.Join(rows, "\n")
}

func (m Model) renderLogs() string {
	rows := []string{"Logs", ""}
	if len(m.logs) == 0 {
		return strings.Join(append(rows, "No operations yet."), "\n")
	}
	return strings.Join(append(rows, m.logs...), "\n")
}

func (m *Model) moveSelection(delta int) {
	count := m.selectionCount()
	if count == 0 {
		return
	}
	m.selected = (m.selected + delta + count) % count
}

func (m Model) selectionCount() int {
	switch m.active {
	case tabLocal:
		return len(m.summary.Skills)
	case tabCatalog:
		return len(m.catalog)
	default:
		return 0
	}
}

func (m *Model) runSelectedCatalogInstall() {
	if m.active != tabCatalog || len(m.catalog) == 0 {
		return
	}
	selected := m.catalog[m.selected]
	m.runOperation(manage.OperationRequest{
		Kind: manage.OperationInstall,
		Spec: selected.Registry + "/" + selected.Identity,
	})
}

func (m *Model) runLocalOperation(kind manage.OperationKind) {
	selected, ok := m.selectedLocalSkill()
	if !ok || selected.Scope != manage.ScopeGlobal {
		return
	}
	request := manage.OperationRequest{Kind: kind, Identity: selected.Identity, Runtime: "codex"}
	m.runOperation(request)
}

func (m *Model) confirmLocalOperation(kind manage.OperationKind) {
	selected, ok := m.selectedLocalSkill()
	if !ok || selected.Scope != manage.ScopeGlobal {
		return
	}
	request := manage.OperationRequest{Kind: kind, Identity: selected.Identity}
	if manage.RequiresConfirmation(request) {
		m.pending = &request
		return
	}
	m.runOperation(request)
}

func (m Model) selectedLocalSkill() (manage.SkillView, bool) {
	if m.active != tabLocal || len(m.summary.Skills) == 0 {
		return manage.SkillView{}, false
	}
	return m.summary.Skills[m.selected], true
}

func (m *Model) runOperation(request manage.OperationRequest) {
	result, err := m.runner(request)
	if err != nil {
		m.logs = append(m.logs, "ERROR "+commandFor(request)+": "+err.Error())
		return
	}
	if result.Command != "" {
		m.logs = append(m.logs, result.Command)
	}
	if result.Message != "" {
		m.logs = append(m.logs, result.Message)
	}
}

func commandFor(request manage.OperationRequest) string {
	switch request.Kind {
	case manage.OperationInstall:
		return "skillhub install " + request.Spec
	case manage.OperationDeploy:
		runtime := request.Runtime
		if runtime == "" {
			runtime = "codex"
		}
		return "skillhub deploy " + runtime + " " + request.Identity
	case manage.OperationRollback:
		return "skillhub rollback " + request.Identity
	case manage.OperationUninstall:
		return "skillhub uninstall " + request.Identity
	case manage.OperationUpdate:
		return "skillhub update"
	default:
		return string(request.Kind)
	}
}

func (m Model) confirmationText() string {
	if m.pending == nil {
		return ""
	}
	switch m.pending.Kind {
	case manage.OperationUninstall:
		return "Confirm uninstall " + m.pending.Identity + "? y/N"
	case manage.OperationRollback:
		return "Confirm rollback " + m.pending.Identity + "? y/N"
	default:
		return "Confirm " + string(m.pending.Kind) + "? y/N"
	}
}

func runtimeSummary(states map[string]string) string {
	runtimes := []string{"codex", "claude", "gemini"}
	var parts []string
	for _, runtime := range runtimes {
		if state, ok := states[runtime]; ok {
			parts = append(parts, runtime+"="+state)
		}
	}
	return strings.Join(parts, " ")
}

func stateOrBlank(states map[string]string, runtime string) string {
	if states == nil {
		return "-"
	}
	if state := states[runtime]; state != "" {
		return state
	}
	return "-"
}
