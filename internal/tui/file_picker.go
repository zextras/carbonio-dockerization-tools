package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/filepicker"
	tea "github.com/charmbracelet/bubbletea"
)

// FilePickerChoiceMsg is sent when user selects a file or cancels
type FilePickerChoiceMsg struct {
	FilePath  string
	Cancelled bool
}

// FilePickerModel represents the file picker screen
type FilePickerModel struct {
	filepicker filepicker.Model
	selected   string
	err        error
}

// NewFilePickerModel creates a new file picker model
func NewFilePickerModel() *FilePickerModel {
	// Get starting directory (current directory or user's home as fallback)
	startDir, err := os.Getwd()
	if err != nil {
		startDir, _ = os.UserHomeDir()
	}

	fp := filepicker.New()
	fp.AllowedTypes = []string{".yaml", ".yml"}
	fp.CurrentDirectory = startDir
	fp.ShowHidden = false
	fp.Height = 15

	return &FilePickerModel{
		filepicker: fp,
	}
}

func (m *FilePickerModel) Init() tea.Cmd {
	return m.filepicker.Init()
}

func (m *FilePickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc", "q":
			// Cancel and go back
			return m, func() tea.Msg {
				return FilePickerChoiceMsg{
					Cancelled: true,
				}
			}
		}

	case tea.WindowSizeMsg:
		m.filepicker.Height = msg.Height - 10
		if m.filepicker.Height < 5 {
			m.filepicker.Height = 5
		}
	}

	// Update filepicker
	var cmd tea.Cmd
	m.filepicker, cmd = m.filepicker.Update(msg)

	// Check if user selected a file
	if didSelect, path := m.filepicker.DidSelectFile(msg); didSelect {
		m.selected = path

		// Validate that it's a .yaml or .yml file
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			m.err = fmt.Errorf("invalid file type: must be .yaml or .yml")
			return m, nil
		}

		// Send selection message
		return m, func() tea.Msg {
			return FilePickerChoiceMsg{
				FilePath:  path,
				Cancelled: false,
			}
		}
	}

	return m, cmd
}

func (m *FilePickerModel) View() string {
	var s strings.Builder

	s.WriteString(titleStyle.Render("📁 Select Configuration File"))
	s.WriteString("\n\n")

	if m.err != nil {
		s.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		s.WriteString("\n\n")
		s.WriteString(helpStyle.Render("press esc to go back"))
		return s.String()
	}

	s.WriteString(m.filepicker.View())
	s.WriteString("\n\n")
	s.WriteString(helpStyle.Render("↑/↓: navigate • enter: select • esc: cancel • q: quit"))

	return s.String()
}
