package tui

import (
	"carbonio-docker-cli/internal/parser"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"os"
	"path/filepath"
	"strings"
)

var (
	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true)
	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00")).
			Bold(true)
)

type ConfigSaveChoiceMsg struct {
	WantsSave bool
	Filename  string
}
type ConfigSaveState int

const (
	StateAsk ConfigSaveState = iota
	StateInput
	StateError
)

type ConfigSaveModel struct {
	edition      parser.Edition
	state        ConfigSaveState
	cursor       int
	choices      []string
	filename     string
	buffer       string
	errorMessage string
}

func NewConfigSaveModel(edition parser.Edition) *ConfigSaveModel {
	return &ConfigSaveModel{
		edition:  edition,
		state:    StateAsk,
		cursor:   0,
		choices:  []string{"Yes", "No"},
		filename: "carbonio-config.yaml",
		buffer:   "carbonio-config.yaml",
	}
}
func (m *ConfigSaveModel) Init() tea.Cmd {
	return nil
}
func (m *ConfigSaveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.state {
	case StateAsk:
		return m.updateAsk(msg)
	case StateInput:
		return m.updateInput(msg)
	case StateError:
		return m.updateError(msg)
	}
	return m, nil
}
func (m *ConfigSaveModel) updateAsk(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if m.cursor == 0 {
				m.state = StateInput
			} else {
				return m, func() tea.Msg {
					return ConfigSaveChoiceMsg{
						WantsSave: false,
						Filename:  "",
					}
				}
			}
		}
	}
	return m, nil
}
func (m *ConfigSaveModel) updateInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.state = StateAsk
			m.buffer = m.filename
			m.errorMessage = ""
		case "enter":
			filename := strings.TrimSpace(m.buffer)
			if filename == "" {
				m.errorMessage = "Filename cannot be empty"
				m.state = StateError
				return m, nil
			}
			exePath, err := os.Executable()
			if err != nil {
				m.errorMessage = fmt.Sprintf("Failed to get binary path: %v", err)
				m.state = StateError
				return m, nil
			}
			binaryDir := filepath.Dir(exePath)
			fullPath := filepath.Join(binaryDir, filename)
			if _, err := os.Stat(fullPath); err == nil {
				m.errorMessage = fmt.Sprintf("File '%s' already exists. Please choose another name.", filename)
				m.state = StateError
				return m, nil
			}
			return m, func() tea.Msg {
				return ConfigSaveChoiceMsg{
					WantsSave: true,
					Filename:  fullPath,
				}
			}
		case "backspace":
			if len(m.buffer) > 0 {
				m.buffer = m.buffer[:len(m.buffer)-1]
			}
		default:
			if len(msg.String()) == 1 {
				m.buffer += msg.String()
			}
		}
	}
	return m, nil
}
func (m *ConfigSaveModel) updateError(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter", "esc":
			m.state = StateInput
			m.errorMessage = ""
		}
	}
	return m, nil
}
func (m *ConfigSaveModel) View() string {
	var s strings.Builder
	s.WriteString(titleStyle.Render("💾 Save Configuration"))
	s.WriteString("\n\n")
	switch m.state {
	case StateAsk:
		s.WriteString("Do you want to save this configuration to a file?\n\n")
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
	case StateInput:
		s.WriteString("Enter filename for configuration:\n\n")
		s.WriteString("Filename: ")
		s.WriteString(inputStyle.Render(m.buffer))
		s.WriteString("█")
		s.WriteString("\n\n")
		s.WriteString(helpStyle.Render("type to edit • enter: save • esc: cancel"))
	case StateError:
		s.WriteString(errorStyle.Render("❌ " + m.errorMessage))
		s.WriteString("\n\n")
		s.WriteString(helpStyle.Render("press enter to try again"))
	}
	return s.String()
}
