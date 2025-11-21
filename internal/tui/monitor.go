package tui

import (
	"carbonio-docker-cli/internal/docker"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"sort"
	"strings"
	"time"

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
		return "📄 Pulling"
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

// DockerStateMsg contains the updated state from docker ps
type DockerStateMsg struct {
	States map[string]ServiceState
}

// MonitorModel represents the live output monitor screen
type MonitorModel struct {
	executor *docker.Executor
	envVars  string
	cmdParts []string
	workDir  string

	outputChan       chan string
	outputLines      []string
	maxLines         int
	done             bool
	err              error
	cleaning         bool
	simpleView       bool // Toggle between simple and detailed view
	selectedServices []string
	serviceStates    map[string]ServiceState

	// Scrolling for simple view
	viewOffset int
	viewHeight int
}

// DockerComposeStatus represents a service status from docker compose ps
type DockerComposeStatus struct {
	Name    string `json:"Name"`
	State   string `json:"State"`
	Status  string `json:"Status"`
	Health  string `json:"Health"`
	Service string `json:"Service"`
}

// NewMonitorModel creates a new monitor model
func NewMonitorModel(executor *docker.Executor, envVars string, cmdParts []string, selectedServices []string, workDir string) *MonitorModel {
	// Initialize service states
	states := make(map[string]ServiceState)
	for _, svc := range selectedServices {
		states[svc] = ServiceStateUnknown
	}

	return &MonitorModel{
		executor:         executor,
		envVars:          envVars,
		cmdParts:         cmdParts,
		workDir:          workDir,
		outputChan:       make(chan string, 100),
		outputLines:      []string{},
		maxLines:         30,
		done:             false,
		cleaning:         false,
		simpleView:       true, // Start with simple view
		selectedServices: selectedServices,
		serviceStates:    states,
		viewOffset:       0,
		viewHeight:       20,
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
	return tea.Batch(
		m.waitForOutput(),
		m.pollDockerState(),
	)
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

// pollDockerState polls docker compose ps to get real container states
func (m *MonitorModel) pollDockerState() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		states := m.fetchDockerStates()
		return DockerStateMsg{States: states}
	})
}

// fetchDockerStates queries docker compose ps to get actual container states
func (m *MonitorModel) fetchDockerStates() map[string]ServiceState {
	states := make(map[string]ServiceState)

	// Run docker compose ps --format json
	cmd := exec.Command("docker", "compose", "-f", "docker-compose.yaml", "-f", "docker-compose-advanced.yaml", "ps", "--format", "json")
	cmd.Dir = m.workDir

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Failed to fetch docker states: %v", err)
		return states
	}

	// Parse JSON output (one JSON object per line)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var status DockerComposeStatus
		if err := json.Unmarshal([]byte(line), &status); err != nil {
			log.Printf("Failed to parse docker status: %v", err)
			continue
		}

		// Map docker state to our ServiceState
		serviceName := status.Service
		if serviceName == "" {
			continue
		}

		// Only track services we're monitoring
		tracked := false
		for _, svc := range m.selectedServices {
			if svc == serviceName {
				tracked = true
				break
			}
		}
		if !tracked {
			continue
		}

		// Map State to ServiceState
		state := strings.ToLower(status.State)
		switch state {
		case "running":
			// Check health if available
			if status.Health == "unhealthy" {
				states[serviceName] = ServiceStateError
			} else {
				states[serviceName] = ServiceStateRunning
			}
		case "created":
			states[serviceName] = ServiceStateCreating
		case "restarting":
			states[serviceName] = ServiceStateStarting
		case "paused":
			states[serviceName] = ServiceStateError
		case "exited":
			// Check exit code from Status field
			if strings.Contains(status.Status, "Exited (0)") {
				// Clean exit - could be a one-shot container
				states[serviceName] = ServiceStateRunning
			} else {
				states[serviceName] = ServiceStateError
			}
		case "dead":
			states[serviceName] = ServiceStateError
		default:
			states[serviceName] = ServiceStateUnknown
		}
	}

	return states
}

func (m *MonitorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewHeight = msg.Height - 10
		if m.viewHeight < 5 {
			m.viewHeight = 5
		}
		log.Printf("Window resized: height=%d, viewHeight=%d", msg.Height, m.viewHeight)
		m.adjustViewOffset()

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

			// Stop and cleanup and then quit
			return m, func() tea.Msg {
				if err := m.executor.StopAndCleanup(); err != nil {
					log.Printf("Failed to stop and cleanup: %v", err)
					return MonitorCompletedMsg{Error: err}
				}
				log.Println("Stop and cleanup successful")
				return MonitorCompletedMsg{Error: nil}
			}

		case "l":
			// Toggle between simple and detailed view
			m.simpleView = !m.simpleView
			log.Printf("Toggled view: simpleView=%v", m.simpleView)

		case "up", "k":
			if m.simpleView && m.viewOffset > 0 {
				m.viewOffset--
				log.Printf("Scrolled up: offset=%d", m.viewOffset)
			}

		case "down", "j":
			if m.simpleView {
				totalServices := len(m.selectedServices)
				maxOffset := totalServices - m.viewHeight
				if maxOffset < 0 {
					maxOffset = 0
				}
				if m.viewOffset < maxOffset {
					m.viewOffset++
					log.Printf("Scrolled down: offset=%d", m.viewOffset)
				}
			}

		case "home", "g":
			if m.simpleView {
				m.viewOffset = 0
				log.Println("Jumped to top")
			}

		case "end", "G":
			if m.simpleView {
				totalServices := len(m.selectedServices)
				m.viewOffset = totalServices - m.viewHeight
				if m.viewOffset < 0 {
					m.viewOffset = 0
				}
				log.Printf("Jumped to bottom: offset=%d", m.viewOffset)
			}
		}

	case DockerStateMsg:
		// Update states from docker ps
		for serviceName, state := range msg.States {
			m.serviceStates[serviceName] = state
		}
		// Continue polling
		if !m.done && !m.cleaning {
			return m, m.pollDockerState()
		}

	case OutputLineMsg:
		// Add line to output
		m.outputLines = append(m.outputLines, msg.Line)

		// Keep only last maxLines
		if len(m.outputLines) > m.maxLines {
			m.outputLines = m.outputLines[len(m.outputLines)-m.maxLines:]
		}

		// Parse line for initial states (before docker ps kicks in)
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
			m.outputLines = append(m.outputLines, "✅ Cleanup complete - Press any key to exit")
		}

		// If we're cleaning up (user initiated shutdown), quit immediately
		if m.cleaning {
			log.Println("Cleanup finished, quitting...")
			return m, tea.Quit
		}

		return m, nil
	}

	return m, nil
}

func (m *MonitorModel) parseLogLine(line string) {
	lineLower := strings.ToLower(line)

	// IGNORA completamente i log interni ai container (righe con pipe)
	if strings.Contains(line, "|") {
		return
	}

	// Parse solo i messaggi di Docker Compose
	for _, serviceName := range m.selectedServices {
		serviceNameLower := strings.ToLower(serviceName)

		// Skip if service not mentioned in this line
		if !strings.Contains(lineLower, serviceNameLower) {
			continue
		}

		// Pattern specifici di Docker Compose (non log interni)
		if strings.Contains(lineLower, "pulling") || strings.Contains(lineLower, "pull") {
			m.serviceStates[serviceName] = ServiceStatePulling
		} else if strings.Contains(lineLower, "creating") {
			m.serviceStates[serviceName] = ServiceStateCreating
		} else if strings.Contains(lineLower, "created") {
			m.serviceStates[serviceName] = ServiceStateCreating
		} else if strings.Contains(lineLower, "starting") {
			m.serviceStates[serviceName] = ServiceStateStarting
		} else if strings.Contains(lineLower, "started") {
			m.serviceStates[serviceName] = ServiceStateRunning
		} else if strings.Contains(lineLower, "error") && !strings.Contains(line, "|") {
			// Solo errori di Docker Compose, non log interni
			m.serviceStates[serviceName] = ServiceStateError
		}
	}
}

func (m *MonitorModel) adjustViewOffset() {
	totalServices := len(m.selectedServices)

	if m.viewOffset < 0 {
		m.viewOffset = 0
	}

	maxOffset := totalServices - m.viewHeight
	if maxOffset < 0 {
		maxOffset = 0
	}

	if m.viewOffset > maxOffset {
		m.viewOffset = maxOffset
	}

	log.Printf("adjustViewOffset: offset=%d, height=%d, total=%d",
		m.viewOffset, m.viewHeight, totalServices)
}

func (m *MonitorModel) View() string {
	var s strings.Builder

	s.WriteString(titleStyle.Render("🚀 Starting Carbonio..."))
	s.WriteString("\n\n")

	if m.simpleView {
		// Simple view: show service states with scrolling
		s.WriteString(sectionStyle.Render("Services Status:"))
		s.WriteString("\n")

		// Sort services alphabetically for consistent display
		sortedServices := make([]string, len(m.selectedServices))
		copy(sortedServices, m.selectedServices)
		sort.Strings(sortedServices)

		totalServices := len(sortedServices)
		startIdx := m.viewOffset
		endIdx := m.viewOffset + m.viewHeight
		if endIdx > totalServices {
			endIdx = totalServices
		}

		for i := startIdx; i < endIdx; i++ {
			serviceName := sortedServices[i]
			state := m.serviceStates[serviceName]
			stateStr := state.String()
			styledState := state.Style().Render(stateStr)

			s.WriteString(fmt.Sprintf("  %s %s\n", styledState, serviceName))
		}

		s.WriteString("\n")
		s.WriteString(helpStyle.Render(fmt.Sprintf("Showing %d-%d of %d services", startIdx+1, endIdx, totalServices)))
		s.WriteString("\n")
		s.WriteString(helpStyle.Render("↑/↓: scroll • g/G: top/bottom • l: toggle detailed logs"))
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
