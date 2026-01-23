package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"strings"
)

type CleanDatabaseChoiceMsg struct {
	CleanDatabase bool
}

type CleanDatabaseModel struct {
	cursor  int
	choices []string
}

func NewCleanDatabaseModel() *CleanDatabaseModel {
	return &CleanDatabaseModel{
		cursor:  1, // Default to "No"
		choices: []string{"Yes", "No"},
	}
}

func (m *CleanDatabaseModel) Init() tea.Cmd {
	return nil
}

func (m *CleanDatabaseModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "left", "h":
			if m.cursor > 0 {
				m.cursor--
			}
		case "right", "l":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter":
			return m, func() tea.Msg {
				return CleanDatabaseChoiceMsg{
					CleanDatabase: m.cursor == 0, // "Yes" is at index 0
				}
			}
		}
	}
	return m, nil
}

func (m *CleanDatabaseModel) View() string {
	var s strings.Builder

	s.WriteString(titleStyle.Render("🗑️  Clean Database"))
	s.WriteString("\n\n")

	s.WriteString("Do you want to start with a clean database?\n\n")
	s.WriteString(helpStyle.Render("This will remove the PostgreSQL volume, deleting all data"))
	s.WriteString("\n")
	s.WriteString(helpStyle.Render("(files, tasks, docs, etc.). Recommended for fresh testing."))
	s.WriteString("\n\n")

	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
			s.WriteString(selectedStyle.Render(fmt.Sprintf("%s [%s]", cursor, choice)))
		} else {
			s.WriteString(optionStyle.Render(fmt.Sprintf("%s [%s]", cursor, choice)))
		}
		s.WriteString("  ")
	}

	s.WriteString("\n\n")
	s.WriteString(helpStyle.Render("←/→: select • enter: confirm • q: quit"))

	return s.String()
}
