package tui

import (
	"carbonio-docker-cli/internal/config"
	"carbonio-docker-cli/internal/graph"
	"carbonio-docker-cli/internal/parser"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"log"
	"sort"
	"strings"
)

var (
	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFA500"))
	checkboxStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))
	disabledStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))
	requiredNameStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#666666"))
	requiredTagStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF"))
	lockedTagStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))
	inputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4"))
)

type ServicesConfirmedMsg struct {
	Backend         map[string]string
	Frontend        map[string]string
	VisibleServices []string
}
type ServiceItem struct {
	ServiceName  string
	ImageBase    string
	DefaultTag   string
	CustomTag    string
	Selected     bool
	IsBackend    bool
	IsRequired   bool
	TagLocked    bool
	Dependencies []string
}
type ServicesModel struct {
	parsedConfig       *parser.ParsedConfig
	resolver           *graph.DependencyResolver
	edition            parser.Edition
	backendItems       []*ServiceItem
	frontendItems      []*ServiceItem
	cursor             int
	editingTag         bool
	editingIndex       int
	editingBuffer      string
	viewOffset         int
	viewHeight         int
	maxBackendNameLen  int
	maxFrontendNameLen int
}

func extractImageBase(imageURL string) string {
	if imageURL == "" {
		return ""
	}
	lastColon := strings.LastIndex(imageURL, ":")
	lastSlash := strings.LastIndex(imageURL, "/")
	if lastColon > lastSlash && lastColon != -1 {
		return imageURL[:lastColon]
	}
	return imageURL
}
func NewServicesModel(parsedConfig *parser.ParsedConfig, resolver *graph.DependencyResolver, edition parser.Edition) *ServicesModel {
	log.Println("=== Creating ServicesModel ===")
	log.Printf("Backend services: %d", len(parsedConfig.BackendServices))
	log.Printf("Frontend images: %d", len(parsedConfig.FrontendImages))
	m := &ServicesModel{
		parsedConfig: parsedConfig,
		resolver:     resolver,
		edition:      edition,
		cursor:       0,
		editingTag:   false,
		editingIndex: -1,
		viewOffset:   0,
		viewHeight:   20,
	}
	var requiredBackend []*ServiceItem
	var optionalBackend []*ServiceItem
	backendNames := make([]string, 0, len(parsedConfig.BackendServices))
	for name := range parsedConfig.BackendServices {
		backendNames = append(backendNames, name)
	}
	sort.Strings(backendNames)
	for _, name := range backendNames {
		svc := parsedConfig.BackendServices[name]
		if parser.GlobalDockerConfig.IsServiceHidden(name) {
			log.Printf("Hiding service from UI: %s", name)
			continue
		}
		imageBase := extractImageBase(svc.DefaultImage)
		if imageBase == "" {
			imageBase = name
			log.Printf("Backend service %s has no image, using service name as fallback", name)
		}
		tagLocked := parser.GlobalDockerConfig.IsTagLocked(name, svc.DefaultTag, true)
		item := &ServiceItem{
			ServiceName:  name,
			ImageBase:    imageBase,
			DefaultTag:   svc.DefaultTag,
			CustomTag:    "",
			Selected:     true,
			IsBackend:    true,
			IsRequired:   svc.IsRequired,
			TagLocked:    tagLocked,
			Dependencies: svc.DependsOn,
		}
		if item.IsRequired {
			requiredBackend = append(requiredBackend, item)
		} else {
			optionalBackend = append(optionalBackend, item)
		}
		if len(name) > m.maxBackendNameLen {
			m.maxBackendNameLen = len(name)
		}
		log.Printf("Backend item: service=%s, image=%s:%s, required=%v, locked=%v",
			name, imageBase, svc.DefaultTag, svc.IsRequired, tagLocked)
	}
	m.backendItems = append(requiredBackend, optionalBackend...)
	var requiredFrontend []*ServiceItem
	var optionalFrontend []*ServiceItem
	frontendNames := make([]string, 0, len(parsedConfig.FrontendImages))
	for name := range parsedConfig.FrontendImages {
		frontendNames = append(frontendNames, name)
	}
	sort.Strings(frontendNames)
	for _, name := range frontendNames {
		ui := parsedConfig.FrontendImages[name]
		imageBase := extractImageBase(ui.DefaultImage)
		tagLocked := parser.GlobalDockerConfig.IsTagLocked(name, ui.DefaultTag, false)
		item := &ServiceItem{
			ServiceName: name,
			ImageBase:   imageBase,
			DefaultTag:  ui.DefaultTag,
			CustomTag:   "",
			Selected:    true,
			IsBackend:   false,
			IsRequired:  ui.IsProxy,
			TagLocked:   tagLocked,
		}
		if item.IsRequired {
			requiredFrontend = append(requiredFrontend, item)
		} else {
			optionalFrontend = append(optionalFrontend, item)
		}
		if len(name) > m.maxFrontendNameLen {
			m.maxFrontendNameLen = len(name)
		}
		log.Printf("Frontend item: ui=%s, image=%s:%s, proxy=%v, locked=%v",
			name, imageBase, ui.DefaultTag, ui.IsProxy, tagLocked)
	}
	m.frontendItems = append(requiredFrontend, optionalFrontend...)
	log.Printf("Created model with %d backend items, %d frontend items",
		len(m.backendItems), len(m.frontendItems))
	log.Printf("Max backend name length: %d, max frontend name length: %d",
		m.maxBackendNameLen, m.maxFrontendNameLen)
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
	case tea.WindowSizeMsg:
		m.viewHeight = msg.Height - 8
		if m.viewHeight < 5 {
			m.viewHeight = 5
		}
		log.Printf("Window resized: height=%d, viewHeight=%d", msg.Height, m.viewHeight)
		m.adjustViewOffset()
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
		case "home", "g":
			m.cursor = 0
			m.viewOffset = 0
			log.Printf("Jump to start: cursor=0, offset=0")
		case "end", "G":
			totalItems := len(m.backendItems) + len(m.frontendItems)
			m.cursor = totalItems - 1
			m.adjustViewOffset()
			log.Printf("Jump to end: cursor=%d", m.cursor)
		case " ":
			m.toggleSelection()
		case "e":
			m.startTagEdit()
		case "enter":
			return m.confirm()
		case "x":
			return m.exportConfig()
		}
	}
	return m, nil
}
func (m *ServicesModel) View() string {
	var s strings.Builder
	s.WriteString(titleStyle.Render("🔧 Select Services and UI Images"))
	s.WriteString("\n")
	editionName := "CE (Community Edition)"
	if m.edition == parser.EditionAdvanced {
		editionName = "Advanced"
	}
	s.WriteString(helpStyle.Render(fmt.Sprintf("Edition: %s", editionName)))
	s.WriteString("\n\n")
	totalItems := len(m.backendItems) + len(m.frontendItems)
	startIdx := m.viewOffset
	endIdx := m.viewOffset + m.viewHeight
	if endIdx > totalItems {
		endIdx = totalItems
	}
	s.WriteString(sectionStyle.Render("Services:"))
	s.WriteString("\n")
	for i := 0; i < len(m.backendItems); i++ {
		if i >= startIdx && i < endIdx {
			s.WriteString(m.renderItem(i, m.backendItems[i], m.maxBackendNameLen))
		}
	}
	s.WriteString("\n")
	s.WriteString(sectionStyle.Render("Composed UI:"))
	s.WriteString("\n")
	for i := 0; i < len(m.frontendItems); i++ {
		globalIndex := len(m.backendItems) + i
		if globalIndex >= startIdx && globalIndex < endIdx {
			s.WriteString(m.renderItem(globalIndex, m.frontendItems[i], m.maxFrontendNameLen))
		}
	}
	s.WriteString("\n")
	if m.editingTag {
		s.WriteString(helpStyle.Render("Editing tag: type to modify • enter: confirm • esc: cancel"))
	} else {
		s.WriteString(helpStyle.Render("↑/↓: navigate • g/G: top/bottom • space: toggle • e: edit tag • enter: start • x: export • q: quit"))
	}
	s.WriteString("\n")
	s.WriteString(helpStyle.Render(fmt.Sprintf("Item %d/%d", m.cursor+1, totalItems)))
	return s.String()
}
func (m *ServicesModel) renderItem(index int, item *ServiceItem, maxNameLen int) string {
	cursor := " "
	if m.cursor == index {
		cursor = ">"
	}
	checkbox := "[ ]"
	if item.Selected {
		checkbox = "[✓]"
	}
	if item.IsRequired {
		checkbox = "[●]"
	}
	tag := item.DefaultTag
	if item.CustomTag != "" {
		tag = item.CustomTag
	}
	fullImageDisplay := fmt.Sprintf("%s:%s", item.ImageBase, tag)
	padding := maxNameLen - len(item.ServiceName) + 2
	if item.IsBackend {
		padding += 10
	} else {
		padding += 15
	}
	paddingStr := strings.Repeat(" ", padding)
	var line string
	if item.IsRequired {
		namePart := requiredNameStyle.Render(fmt.Sprintf("%s %s %s", cursor, checkbox, item.ServiceName))
		var imagePart string
		if item.TagLocked {
			imagePart = lockedTagStyle.Render(fmt.Sprintf("(%s)", fullImageDisplay))
		} else {
			imagePart = requiredTagStyle.Render(fmt.Sprintf("(%s)", fullImageDisplay))
		}
		line = namePart + paddingStr + imagePart
	} else {
		if item.TagLocked {
			namePart := fmt.Sprintf("%s %s %s", cursor, checkbox, item.ServiceName)
			imagePart := lockedTagStyle.Render(fmt.Sprintf("(%s)", fullImageDisplay))
			line = namePart + paddingStr + imagePart
		} else {
			line = fmt.Sprintf("%s %s %s%s(%s)", cursor, checkbox, item.ServiceName, paddingStr, fullImageDisplay)
		}
	}
	if m.editingTag && m.editingIndex == index {
		return selectedStyle.Render(line) + " ← " + inputStyle.Render(m.editingBuffer) + "\n"
	}
	if m.cursor == index {
		return selectedStyle.Render(line) + "\n"
	}
	return checkboxStyle.Render(line) + "\n"
}
func (m *ServicesModel) toggleSelection() {
	item := m.getCurrentItem()
	if item == nil || item.IsRequired {
		return
	}
	item.Selected = !item.Selected
	log.Printf("Toggled service %s: selected=%v", item.ServiceName, item.Selected)
	if item.IsBackend && item.Selected {
		m.autoSelectDependencies(item)
	}
}
func (m *ServicesModel) autoSelectDependencies(item *ServiceItem) {
	deps := m.resolver.ResolveDependencies(item.ServiceName)
	log.Printf("Auto-selecting dependencies for %s: %v", item.ServiceName, deps)
	for _, depName := range deps {
		for _, backendItem := range m.backendItems {
			if backendItem.ServiceName == depName {
				backendItem.Selected = true
				log.Printf("  - Selected dependency: %s", depName)
			}
		}
	}
}
func (m *ServicesModel) startTagEdit() {
	item := m.getCurrentItem()
	if item == nil || item.TagLocked {
		log.Printf("Cannot edit tag for %s: tag is locked", item.ServiceName)
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
			item := m.getCurrentItem()
			if item != nil && !item.TagLocked {
				item.CustomTag = m.editingBuffer
			}
			m.editingTag = false
			m.editingIndex = -1
			m.editingBuffer = ""
		case "esc":
			m.editingTag = false
			m.editingIndex = -1
			m.editingBuffer = ""
		case "backspace":
			if len(m.editingBuffer) > 0 {
				m.editingBuffer = m.editingBuffer[:len(m.editingBuffer)-1]
			}
		default:
			if len(msg.String()) == 1 {
				m.editingBuffer += msg.String()
			}
		}
	}
	return m, nil
}
func (m *ServicesModel) confirm() (tea.Model, tea.Cmd) {
	log.Println("=== Confirm called ===")
	backend := make(map[string]string)
	frontend := make(map[string]string)
	visibleServices := []string{}
	log.Printf("Building backend selection from %d visible items", len(m.backendItems))
	for _, item := range m.backendItems {
		if item.Selected {
			tag := item.DefaultTag
			if item.CustomTag != "" && !item.TagLocked {
				tag = item.CustomTag
			}
			backend[item.ServiceName] = tag
			visibleServices = append(visibleServices, item.ServiceName)
			log.Printf("  Backend: %s -> %s", item.ServiceName, tag)
			registrators := parser.GlobalDockerConfig.GetRegistratorsForService(item.ServiceName)
			for _, regName := range registrators {
				if regSvc, exists := m.parsedConfig.BackendServices[regName]; exists {
					backend[regName] = regSvc.DefaultTag
					log.Printf("  Auto-added registrator: %s -> %s (for %s)",
						regName, regSvc.DefaultTag, item.ServiceName)
				}
			}
		}
	}
	for _, autoIncludedName := range parser.GlobalDockerConfig.AutoIncludedServices {
		if autoIncludedSvc, exists := m.parsedConfig.BackendServices[autoIncludedName]; exists {
			backend[autoIncludedName] = autoIncludedSvc.DefaultTag
			log.Printf("  Auto-added hidden service: %s -> %s", autoIncludedName, autoIncludedSvc.DefaultTag)
		}
	}
	log.Printf("Building frontend selection from %d visible items", len(m.frontendItems))
	for _, item := range m.frontendItems {
		tag := item.DefaultTag
		if item.CustomTag != "" && !item.TagLocked {
			tag = item.CustomTag
		}
		if !item.Selected {
			tag = "disabled"
		}
		frontend[item.ServiceName] = tag
		log.Printf("  Frontend: %s -> %s", item.ServiceName, tag)
	}
	log.Printf("Sending confirmation message with %d backend, %d frontend, %d visible services",
		len(backend), len(frontend), len(visibleServices))
	return m, func() tea.Msg {
		return ServicesConfirmedMsg{
			Backend:         backend,
			Frontend:        frontend,
			VisibleServices: visibleServices,
		}
	}
}
func (m *ServicesModel) exportConfig() (tea.Model, tea.Cmd) {
	backend := make(map[string]string)
	frontend := make(map[string]string)
	for _, item := range m.backendItems {
		if item.Selected {
			tag := item.DefaultTag
			if item.CustomTag != "" && !item.TagLocked {
				tag = item.CustomTag
			}
			backend[item.ServiceName] = tag
			registrators := parser.GlobalDockerConfig.GetRegistratorsForService(item.ServiceName)
			for _, regName := range registrators {
				if regSvc, exists := m.parsedConfig.BackendServices[regName]; exists {
					backend[regName] = regSvc.DefaultTag
				}
			}
		}
	}
	for _, autoIncludedName := range parser.GlobalDockerConfig.AutoIncludedServices {
		if autoIncludedSvc, exists := m.parsedConfig.BackendServices[autoIncludedName]; exists {
			backend[autoIncludedName] = autoIncludedSvc.DefaultTag
		}
	}
	for _, item := range m.frontendItems {
		tag := item.DefaultTag
		if item.CustomTag != "" && !item.TagLocked {
			tag = item.CustomTag
		}
		if !item.Selected {
			tag = "disabled"
		}
		frontend[item.ServiceName] = tag
	}
	userConfig := config.CreateUserConfig(string(m.edition), backend, frontend)
	filename := "carbonio-config.yaml"
	if err := config.ExportConfig(filename, userConfig); err != nil {
		return m, nil
	}
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
	if m.viewHeight <= 0 {
		m.viewHeight = 20
	}
	totalItems := len(m.backendItems) + len(m.frontendItems)
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= totalItems {
		m.cursor = totalItems - 1
	}
	if m.cursor < m.viewOffset {
		m.viewOffset = m.cursor
	}
	if m.cursor >= m.viewOffset+m.viewHeight {
		m.viewOffset = m.cursor - m.viewHeight + 1
	}
	if m.viewOffset < 0 {
		m.viewOffset = 0
	}
	maxOffset := totalItems - m.viewHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.viewOffset > maxOffset {
		m.viewOffset = maxOffset
	}
	log.Printf("adjustViewOffset: cursor=%d, offset=%d, height=%d, total=%d",
		m.cursor, m.viewOffset, m.viewHeight, totalItems)
}
