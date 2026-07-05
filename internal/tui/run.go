package tui

import (
	"github.com/CassianFlorin/skill-hub/internal/manage"
	tea "github.com/charmbracelet/bubbletea"
)

func Run(workDir string) error {
	summary, err := manage.Summary(workDir)
	if err != nil {
		return err
	}
	catalog, err := manage.SearchCatalog(workDir, "")
	if err != nil {
		return err
	}
	model := NewModel(summary, catalog).WithRunner(func(request manage.OperationRequest) (manage.OperationResult, error) {
		return manage.Execute(workDir, request)
	})
	program := tea.NewProgram(model, tea.WithAltScreen())
	_, err = program.Run()
	return err
}
