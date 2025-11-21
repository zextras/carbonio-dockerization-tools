package tui

import (
	"carbonio-docker-cli/internal/docker"
	"encoding/json"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"log"
	"os/exec"
	"sort"
	"strings"
	"time"
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

type MonitorCompletedMsg struct {
	Error error
}
type OutputLineMsg struct {
	Line string
}
type DockerStateMsg struct {
	States map[string]ServiceState
}
type MonitorModel struct {
	executor         *docker.Executor
	envVars          string
	cmdParts         []string
	workDir          string
	outputChan       chan string
	outputLines      []string
	maxLines         int
	done             bool
	err              error
	cleaning         bool
	simpleView       bool
	selectedServices []string
	serviceStates    map[string]ServiceState
	viewOffset       int
	viewHeight       int
}
type DockerComposeStatus struct {
	Name    string `json:"Name"`
	State   string `json:"State"`
	Status  string `json:"Status"`
	Health  string `json:"Health"`
	Service string `json:"Service"`
}

func NewMonitorModel(executor *docker.Executor, envVars string, cmdParts []string, selectedServices []string, workDir string) *MonitorModel {
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
		simpleView:       true,
		selectedServices: selectedServices,
		serviceStates:    states,
		viewOffset:       0,
		viewHeight:       20,
	}
}
func (m *MonitorModel) Init() tea.Cmd {
	return nil
}
func (m *MonitorModel) Start() tea.Cmd {
	log.Println("=== MonitorModel.Start() called ===")
	go func() {
		log.Println("Starting docker compose execution...")
		err := m.executor.Execute(m.envVars, m.cmdParts, m.outputChan)
		if err != nil {
			log.Printf("Docker execution failed: %v", err)
			m.err = err
		}
		log.Println("Docker compose execution finished")
	}()
	return tea.Batch(
		m.waitForOutput(),
		m.pollDockerState(),
	)
}
func (m *MonitorModel) waitForOutput() tea.Cmd {
	return func() tea.Msg {
		line, ok := <-m.outputChan
		if !ok {
			log.Println("Output channel closed")
			return MonitorCompletedMsg{Error: m.err}
		}
		return OutputLineMsg{Line: line}
	}
}
func (m *MonitorModel) pollDockerState() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		states := m.fetchDockerStates()
		return DockerStateMsg{States: states}
	})
}
func (m *MonitorModel) fetchDockerStates() map[string]ServiceState {
	states := make(map[string]ServiceState)
	cmd := exec.Command("docker", "compose", "-f", "docker-compose.yaml", "-f", "docker-compose-advanced.yaml", "ps", "--format", "json")
	cmd.Dir = m.workDir
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Failed to fetch docker states: %v", err)
		return states
	}
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
		serviceName := status.Service
		if serviceName == "" {
			continue
		}
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
		state := strings.ToLower(status.State)
		switch state {
		case "running":
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
			if strings.Contains(status.Status, "Exited (0)") {
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
				return m, nil
			}
			log.Println("User pressed ctrl+c or q, stopping and cleaning up...")
			m.cleaning = true
			m.outputLines = append(m.outputLines, "")
			m.outputLines = append(m.outputLines, "🧹 Stopping and cleaning up containers...")
			return m, func() tea.Msg {
				if err := m.executor.StopAndCleanup(); err != nil {
					log.Printf("Failed to stop and cleanup: %v", err)
					return MonitorCompletedMsg{Error: err}
				}
				log.Println("Stop and cleanup successful")
				return MonitorCompletedMsg{Error: nil}
			}
		case "l":
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
		for serviceName, state := range msg.States {
			m.serviceStates[serviceName] = state
		}
		if !m.done && !m.cleaning {
			return m, m.pollDockerState()
		}
	case OutputLineMsg:
		m.outputLines = append(m.outputLines, msg.Line)
		if len(m.outputLines) > m.maxLines {
			m.outputLines = m.outputLines[len(m.outputLines)-m.maxLines:]
		}
		m.parseLogLine(msg.Line)
		return m, m.waitForOutput()
	case MonitorCompletedMsg:
		log.Println("MonitorCompletedMsg received")
		m.done = true
		if msg.Error != nil {
			log.Printf("Docker completed with error: %v", msg.Error)
			m.err = msg.Error
			m.outputLines = append(m.outputLines, "")
			m.outputLines = append(m.outputLines, fmt.Sprintf("❌ Error: %v", msg.Error))
		} else {
			log.Println("Docker completed successfully")
			m.outputLines = append(m.outputLines, "")
			m.outputLines = append(m.outputLines, "✅ Cleanup complete - Press any key to exit")
		}
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
	if strings.Contains(line, "|") {
		return
	}
	for _, serviceName := range m.selectedServices {
		serviceNameLower := strings.ToLower(serviceName)
		if !strings.Contains(lineLower, serviceNameLower) {
			continue
		}
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
		s.WriteString(sectionStyle.Render("Services Status:"))
		s.WriteString("\n")
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
