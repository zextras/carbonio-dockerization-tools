package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")).
			MarginTop(1).
			MarginBottom(1)
	optionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))
	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")).
			Background(lipgloss.Color("#3C3C3C"))
	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			MarginTop(2)
)

type StartupChoiceMsg struct {
	ImportConfig bool
}
type StartupModel struct {
	cursor  int
	choices []string
}

func NewStartupModel() *StartupModel {
	return &StartupModel{
		cursor: 0,
		choices: []string{
			"Import configuration from file",
			"Custom configuration",
		},
	}
}
func (m *StartupModel) Init() tea.Cmd {
	return nil
}
func (m *StartupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter":
			return m, func() tea.Msg {
				return StartupChoiceMsg{
					ImportConfig: m.cursor == 0,
				}
			}
		}
	}
	return m, nil
}
func (m *StartupModel) View() string {
	s := titleStyle.Render("🚀 Carbonio Docker CLI")
	s += "\n\n"
	s += "Select startup mode:\n\n"
	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
			s += selectedStyle.Render(fmt.Sprintf("%s %s", cursor, choice))
		} else {
			s += optionStyle.Render(fmt.Sprintf("%s %s", cursor, choice))
		}
		s += "\n"
	}
	s += helpStyle.Render("\n↑/↓: navigate • enter: select • q: quit")
	return s
}
