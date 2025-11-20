package tui

import (
	"carbonio-docker-cli/internal/docker"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	outputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000"))
)

// MonitorCompletedMsg is sent when docker compose finishes
type MonitorCompletedMsg struct{}

// OutputLineMsg contains a line of output from docker compose
type OutputLineMsg struct {
	Line string
}

// MonitorModel represents the live output monitor screen
type MonitorModel struct {
	executor *docker.Executor
	envVars  string
	cmdParts []string

	outputLines []string
	maxLines    int
	done        bool
}

// NewMonitorModel creates a new monitor model
func NewMonitorModel(executor *docker.Executor, envVars string, cmdParts []string) *MonitorModel {
	return &MonitorModel{
		executor:    executor,
		envVars:     envVars,
		cmdParts:    cmdParts,
		outputLines: []string{},
		maxLines:    30,
		done:        false,
	}
}

func (m *MonitorModel) Init() tea.Cmd {
	return nil
}

func (m *MonitorModel) Start() tea.Cmd {
	return func() tea.Msg {
		outputChan := make(chan string, 100)

		// Start goroutine to convert output to messages
		go func() {
			for line := range outputChan {
				// Send line as message (in real app, use tea.Batch)
				fmt.Println(line) // Temporary - should use proper message passing
			}
		}()

		// Execute docker compose
		err := m.executor.Execute(m.envVars, m.cmdParts, outputChan)

		if err != nil {
			return MonitorCompletedMsg{}
		}

		return MonitorCompletedMsg{}
	}
}

func (m *MonitorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			// Stop docker compose
			m.executor.Stop()
			return m, tea.Quit
		}

	case OutputLineMsg:
		// Add line to output
		m.outputLines = append(m.outputLines, msg.Line)

		// Keep only last maxLines
		if len(m.outputLines) > m.maxLines {
			m.outputLines = m.outputLines[len(m.outputLines)-m.maxLines:]
		}

	case MonitorCompletedMsg:
		m.done = true
		return m, tea.Quit
	}

	return m, nil
}

func (m *MonitorModel) View() string {
	var s strings.Builder

	s.WriteString(titleStyle.Render("🚀 Starting Carbonio..."))
	s.WriteString("\n\n")

	// Show output lines
	for _, line := range m.outputLines {
		if strings.Contains(line, "[ERROR]") {
			s.WriteString(errorStyle.Render(line))
		} else {
			s.WriteString(outputStyle.Render(line))
		}
		s.WriteString("\n")
	}

	if !m.done {
		s.WriteString("\n")
		s.WriteString(helpStyle.Render("ctrl+c: stop and exit"))
	} else {
		s.WriteString("\n")
		s.WriteString(helpStyle.Render("Press any key to exit"))
	}

	return s.String()
}
