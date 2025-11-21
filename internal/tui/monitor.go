package tui

import (
	"carbonio-docker-cli/internal/docker"
	"fmt"
	"log"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	outputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))

	serviceReadyStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00FF00"))

	serviceWaitingStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFA500"))

	serviceErrorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF0000"))
)

// ServiceState represents the state of a service
type ServiceState int

const (
	ServiceStateUnknown ServiceState = iota
	ServiceStatePulling
	ServiceStateCreating
	ServiceStateStarting
	ServiceStateRunning
	ServiceStateError
)

func (s ServiceState) String() string {
	switch s {
	case ServiceStatePulling:
		return "🔄 Pulling"
	case ServiceStateCreating:
		return "🔨 Creating"
	case ServiceStateStarting:
		return "⏳ Starting"
	case ServiceStateRunning:
		return "✓ Running"
	case ServiceStateError:
		return "❌ Error"
	default:
		return "⏳ Waiting"
	}
}

func (s ServiceState) Style() lipgloss.Style {
	switch s {
	case ServiceStateRunning:
		return serviceReadyStyle
	case ServiceStateError:
		return serviceErrorStyle
	default:
		return serviceWaitingStyle
	}
}

// MonitorCompletedMsg is sent when docker compose finishes
type MonitorCompletedMsg struct {
	Error error
}

// OutputLineMsg contains a line of output from docker compose
type OutputLineMsg struct {
	Line string
}

// MonitorModel represents the live output monitor screen
type MonitorModel struct {
	executor *docker.Executor
	envVars  string
	cmdParts []string

	outputChan       chan string
	outputLines      []string
	maxLines         int
	done             bool
	err              error
	cleaning         bool
	simpleView       bool // Toggle between simple and detailed view
	selectedServices []string
	serviceStates    map[string]ServiceState
}

// NewMonitorModel creates a new monitor model
func NewMonitorModel(executor *docker.Executor, envVars string, cmdParts []string, selectedServices []string) *MonitorModel {
	// Initialize service states
	states := make(map[string]ServiceState)
	for _, svc := range selectedServices {
		states[svc] = ServiceStateUnknown
	}

	return &MonitorModel{
		executor:         executor,
		envVars:          envVars,
		cmdParts:         cmdParts,
		outputChan:       make(chan string, 100),
		outputLines:      []string{},
		maxLines:         30,
		done:             false,
		cleaning:         false,
		simpleView:       true, // Start with simple view
		selectedServices: selectedServices,
		serviceStates:    states,
	}
}

func (m *MonitorModel) Init() tea.Cmd {
	return nil
}

// Start initiates docker compose execution asynchronously
func (m *MonitorModel) Start() tea.Cmd {
	log.Println("=== MonitorModel.Start() called ===")

	// Start docker compose in a goroutine
	go func() {
		log.Println("Starting docker compose execution...")
		err := m.executor.Execute(m.envVars, m.cmdParts, m.outputChan)
		if err != nil {
			log.Printf("Docker execution failed: %v", err)
			m.err = err
		}
		log.Println("Docker compose execution finished")
	}()

	// Start reading from output channel
	return m.waitForOutput()
}

// waitForOutput returns a Cmd that waits for the next line of output
func (m *MonitorModel) waitForOutput() tea.Cmd {
	return func() tea.Msg {
		line, ok := <-m.outputChan
		if !ok {
			// Channel closed - docker compose finished
			log.Println("Output channel closed")
			return MonitorCompletedMsg{Error: m.err}
		}
		return OutputLineMsg{Line: line}
	}
}

func (m *MonitorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.cleaning {
				// Already cleaning, ignore
				return m, nil
			}

			log.Println("User pressed ctrl+c or q, stopping and cleaning up...")
			m.cleaning = true
			m.outputLines = append(m.outputLines, "")
			m.outputLines = append(m.outputLines, "🧹 Stopping and cleaning up containers...")

			// Stop and cleanup in a goroutine
			go func() {
				if err := m.executor.StopAndCleanup(); err != nil {
					log.Printf("Failed to stop and cleanup: %v", err)
					m.err = err
					m.outputLines = append(m.outputLines, fmt.Sprintf("Warning: Cleanup failed: %v", err))
				} else {
					log.Println("Stop and cleanup successful")
					m.outputLines = append(m.outputLines, "✓ Cleanup complete")
				}
				m.done = true
			}()

			return m, nil

		case "l":
			// Toggle between simple and detailed view
			m.simpleView = !m.simpleView
			log.Printf("Toggled view: simpleView=%v", m.simpleView)
		}

	case OutputLineMsg:
		// Add line to output
		m.outputLines = append(m.outputLines, msg.Line)

		// Keep only last maxLines
		if len(m.outputLines) > m.maxLines {
			m.outputLines = m.outputLines[len(m.outputLines)-m.maxLines:]
		}

		// Parse line to update service states
		m.parseLogLine(msg.Line)

		// Continue reading from channel
		return m, m.waitForOutput()

	case MonitorCompletedMsg:
		log.Println("MonitorCompletedMsg received")
		m.done = true
		if msg.Error != nil {
			log.Printf("Docker completed with error: %v", msg.Error)
			m.err = msg.Error
			// Add error to output
			m.outputLines = append(m.outputLines, "")
			m.outputLines = append(m.outputLines, fmt.Sprintf("❌ Error: %v", msg.Error))
		} else {
			log.Println("Docker completed successfully")
			// Add success message
			m.outputLines = append(m.outputLines, "")
			m.outputLines = append(m.outputLines, "✅ Docker Compose completed successfully")
		}
		return m, nil
	}

	return m, nil
}

func (m *MonitorModel) parseLogLine(line string) {
	lineLower := strings.ToLower(line)

	// Check for each tracked service
	for _, serviceName := range m.selectedServices {
		serviceNameLower := strings.ToLower(serviceName)

		// Skip if service not mentioned in this line
		if !strings.Contains(lineLower, serviceNameLower) {
			continue
		}

		// Detect state based on keywords
		if strings.Contains(lineLower, "pulling") || strings.Contains(lineLower, "pull") {
			m.serviceStates[serviceName] = ServiceStatePulling
		} else if strings.Contains(lineLower, "creating") || strings.Contains(lineLower, "created") {
			m.serviceStates[serviceName] = ServiceStateCreating
		} else if strings.Contains(lineLower, "starting") || strings.Contains(lineLower, "started") {
			m.serviceStates[serviceName] = ServiceStateStarting
		} else if strings.Contains(lineLower, "error") || strings.Contains(lineLower, "failed") || strings.Contains(lineLower, "exited") {
			m.serviceStates[serviceName] = ServiceStateError
		} else if strings.Contains(line, "|") {
			// Container is emitting logs (format: "service-1 | log message")
			// This means it's running
			m.serviceStates[serviceName] = ServiceStateRunning
		} else if strings.Contains(lineLower, "running") || strings.Contains(lineLower, "healthy") {
			m.serviceStates[serviceName] = ServiceStateRunning
		}
	}
}

func (m *MonitorModel) View() string {
	var s strings.Builder

	s.WriteString(titleStyle.Render("🚀 Starting Carbonio..."))
	s.WriteString("\n\n")

	if m.simpleView {
		// Simple view: show service states
		s.WriteString(sectionStyle.Render("Services Status:"))
		s.WriteString("\n")

		for _, serviceName := range m.selectedServices {
			state := m.serviceStates[serviceName]
			stateStr := state.String()
			styledState := state.Style().Render(stateStr)

			s.WriteString(fmt.Sprintf("  %s %s\n", styledState, serviceName))
		}

		s.WriteString("\n")
		s.WriteString(helpStyle.Render("l: toggle detailed logs"))
	} else {
		// Detailed view: show all logs
		for _, line := range m.outputLines {
			s.WriteString(outputStyle.Render(line))
			s.WriteString("\n")
		}

		s.WriteString("\n")
		s.WriteString(helpStyle.Render("l: toggle simple view"))
	}

	s.WriteString(" • ")
	if m.cleaning {
		s.WriteString(helpStyle.Render("Cleaning up... Please wait"))
	} else if !m.done {
		s.WriteString(helpStyle.Render("ctrl+c/q: stop and cleanup"))
	} else {
		if m.err != nil {
			s.WriteString(helpStyle.Render("Press ctrl+c or q to exit"))
		} else {
			s.WriteString(helpStyle.Render("Press ctrl+c or q to stop and cleanup"))
		}
	}

	return s.String()
}
