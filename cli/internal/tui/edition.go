package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zextras/carbonio-base-dockerization/cli/internal/parser"
)

// EditionChoiceMsg is sent when user selects an edition
type EditionChoiceMsg struct {
	Edition parser.Edition
}

// EditionModel represents the edition selection screen
type EditionModel struct {
	cursor  int
	choices []string
}

// NewEditionModel creates a new edition model
func NewEditionModel() *EditionModel {
	return &EditionModel{
		cursor: 0,
		choices: []string{
			"CE (Community Edition)",
			"Advanced",
		},
	}
}

func (m *EditionModel) Init() tea.Cmd {
	return nil
}

func (m *EditionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			edition := parser.EditionCE
			if m.cursor == 1 {
				edition = parser.EditionAdvanced
			}

			return m, func() tea.Msg {
				return EditionChoiceMsg{Edition: edition}
			}
		}
	}

	return m, nil
}

func (m *EditionModel) View() string {
	s := titleStyle.Render("📦 Select Carbonio Edition")
	s += "\n\n"

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
