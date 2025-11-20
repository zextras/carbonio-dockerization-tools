package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zextras/carbonio-base-dockerization/cli/internal/config"
	"github.com/zextras/carbonio-base-dockerization/cli/internal/graph"
	"github.com/zextras/carbonio-base-dockerization/cli/internal/parser"
)

var (
	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFA500")).
			MarginTop(1).
			MarginBottom(1)

	checkboxStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))

	inputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4"))
)

// ServicesConfirmedMsg is sent when user confirms selections
type ServicesConfirmedMsg struct {
	Backend  map[string]string // service_name: tag
	Frontend map[string]string // ui_name: tag
}

// ServiceItem represents a selectable service
type ServiceItem struct {
	Name         string
	DefaultTag   string
	CustomTag    string
	Selected     bool
	IsBackend    bool
	Dependencies []string
}

// ServicesModel represents the service selection screen
type ServicesModel struct {
	parsedConfig *parser.ParsedConfig
	resolver     *graph.DependencyResolver
	edition      parser.Edition

	backendItems  []*ServiceItem
	frontendItems []*ServiceItem

	cursor        int
	editingTag    bool
	editingIndex  int
	editingBuffer string

	viewOffset int
}

// NewServicesModel creates a new services model
func NewServicesModel(parsedConfig *parser.ParsedConfig, resolver *graph.DependencyResolver, edition parser.Edition) *ServicesModel {
	m := &ServicesModel{
		parsedConfig: parsedConfig,
		resolver:     resolver,
		edition:      edition,
		cursor:       0,
		editingTag:   false,
		editingIndex: -1,
	}

	// Create backend items (sorted)
	backendNames := make([]string, 0, len(parsedConfig.BackendServices))
	for name := range parsedConfig.BackendServices {
		backendNames = append(backendNames, name)
	}
	sort.Strings(backendNames)

	for _, name := range backendNames {
		svc := parsedConfig.BackendServices[name]
		m.backendItems = append(m.backendItems, &ServiceItem{
			Name:         name,
			DefaultTag:   svc.DefaultTag,
			CustomTag:    "",
			Selected:     true, // All selected by default
			IsBackend:    true,
			Dependencies: svc.DependsOn,
		})
	}

	// Create frontend items (sorted)
	frontendNames := make([]string, 0, len(parsedConfig.FrontendImages))
	for name := range parsedConfig.FrontendImages {
		frontendNames = append(frontendNames, name)
	}
	sort.Strings(frontendNames)

	for _, name := range frontendNames {
		ui := parsedConfig.FrontendImages[name]
		m.frontendItems = append(m.frontendItems, &ServiceItem{
			Name:       name,
			DefaultTag: ui.DefaultTag,
			CustomTag:  "",
			Selected:   true, // All selected by default
			IsBackend:  false,
		})
	}

	return m
}

func (m *ServicesModel) Init() tea.Cmd {
	return nil
}

func (m *ServicesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.editingTag {
		return m.handleTagEdit(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.adjustViewOffset()
			}

		case "down", "j":
			totalItems := len(m.backendItems) + len(m.frontendItems)
			if m.cursor < totalItems-1 {
				m.cursor++
				m.adjustViewOffset()
			}

		case " ":
			// Toggle selection
			m.toggleSelection()

		case "e":
			// Edit tag
			m.startTagEdit()

		case "enter":
			// Confirm and proceed
			return m.confirm()

		case "x":
			// Export config
			return m.exportConfig()
		}
	}

	return m, nil
}

func (m *ServicesModel) View() string {
	var s strings.Builder

	s.WriteString(titleStyle.Render("🔧 Select Services and UI Images"))
	s.WriteString("\n\n")

	// Backend section
	s.WriteString(sectionStyle.Render("Backend Services:"))
	s.WriteString("\n")

	for i, item := range m.backendItems {
		s.WriteString(m.renderItem(i, item))
	}

	s.WriteString("\n")
	s.WriteString(sectionStyle.Render("Frontend UI Images:"))
	s.WriteString("\n")

	for i, item := range m.frontendItems {
		s.WriteString(m.renderItem(len(m.backendItems)+i, item))
	}

	s.WriteString("\n")
	s.WriteString(helpStyle.Render("↑/↓: navigate • space: toggle • e: edit tag • enter: start • x: export config • q: quit"))

	return s.String()
}

func (m *ServicesModel) renderItem(index int, item *ServiceItem) string {
	cursor := " "
	if m.cursor == index {
		cursor = ">"
	}

	checkbox := "[ ]"
	if item.Selected {
		checkbox = "[✓]"
	}

	tag := item.DefaultTag
	if item.CustomTag != "" {
		tag = item.CustomTag
	}

	line := fmt.Sprintf("%s %s %s (%s)", cursor, checkbox, item.Name, tag)

	if m.cursor == index {
		return selectedStyle.Render(line) + "\n"
	}

	return checkboxStyle.Render(line) + "\n"
}

func (m *ServicesModel) toggleSelection() {
	item := m.getCurrentItem()
	if item == nil {
		return
	}

	item.Selected = !item.Selected

	// If backend service, handle dependencies
	if item.IsBackend && item.Selected {
		m.autoSelectDependencies(item)
	}
}

func (m *ServicesModel) autoSelectDependencies(item *ServiceItem) {
	// Get all dependencies
	deps := m.resolver.ResolveDependencies(item.Name)

	// Auto-select all dependencies
	for _, depName := range deps {
		for _, backendItem := range m.backendItems {
			if backendItem.Name == depName {
				backendItem.Selected = true
			}
		}
	}
}

func (m *ServicesModel) startTagEdit() {
	item := m.getCurrentItem()
	if item == nil || !item.Selected {
		return
	}

	m.editingTag = true
	m.editingIndex = m.cursor

	if item.CustomTag != "" {
		m.editingBuffer = item.CustomTag
	} else {
		m.editingBuffer = item.DefaultTag
	}
}

func (m *ServicesModel) handleTagEdit(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			// Confirm tag edit
			item := m.getCurrentItem()
			if item != nil {
				item.CustomTag = m.editingBuffer
			}
			m.editingTag = false
			m.editingIndex = -1
			m.editingBuffer = ""

		case "esc":
			// Cancel tag edit
			m.editingTag = false
			m.editingIndex = -1
			m.editingBuffer = ""

		case "backspace":
			if len(m.editingBuffer) > 0 {
				m.editingBuffer = m.editingBuffer[:len(m.editingBuffer)-1]
			}

		default:
			// Add character to buffer
			if len(msg.String()) == 1 {
				m.editingBuffer += msg.String()
			}
		}
	}

	return m, nil
}

func (m *ServicesModel) confirm() (tea.Model, tea.Cmd) {
	// Check that at least one service is selected
	hasSelected := false
	for _, item := range m.backendItems {
		if item.Selected {
			hasSelected = true
			break
		}
	}

	if !hasSelected {
		// TODO: Show error message
		return m, nil
	}

	// Build selection maps
	backend := make(map[string]string)
	frontend := make(map[string]string)

	for _, item := range m.backendItems {
		if item.Selected {
			tag := item.DefaultTag
			if item.CustomTag != "" {
				tag = item.CustomTag
			}
			backend[item.Name] = tag
		}
	}

	for _, item := range m.frontendItems {
		if item.Selected {
			tag := item.DefaultTag
			if item.CustomTag != "" {
				tag = item.CustomTag
			}
			frontend[item.Name] = tag
		}
	}

	return m, func() tea.Msg {
		return ServicesConfirmedMsg{
			Backend:  backend,
			Frontend: frontend,
		}
	}
}

func (m *ServicesModel) exportConfig() (tea.Model, tea.Cmd) {
	// Build selection maps
	backend := make(map[string]string)
	frontend := make(map[string]string)

	for _, item := range m.backendItems {
		if item.Selected {
			tag := item.DefaultTag
			if item.CustomTag != "" {
				tag = item.CustomTag
			}
			backend[item.Name] = tag
		}
	}

	for _, item := range m.frontendItems {
		if item.Selected {
			tag := item.DefaultTag
			if item.CustomTag != "" {
				tag = item.CustomTag
			}
			frontend[item.Name] = tag
		}
	}

	// Create config
	userConfig := config.CreateUserConfig(string(m.edition), backend, frontend)

	// Export to file
	filename := "carbonio-config.yaml"
	if err := config.ExportConfig(filename, userConfig); err != nil {
		// TODO: Show error
		return m, nil
	}

	// TODO: Show success message
	fmt.Printf("\n✓ Configuration exported to %s\n", filename)

	return m, nil
}

func (m *ServicesModel) getCurrentItem() *ServiceItem {
	if m.cursor < len(m.backendItems) {
		return m.backendItems[m.cursor]
	}

	frontendIndex := m.cursor - len(m.backendItems)
	if frontendIndex < len(m.frontendItems) {
		return m.frontendItems[frontendIndex]
	}

	return nil
}

func (m *ServicesModel) adjustViewOffset() {
	// TODO: Implement scrolling if list is too long
}
