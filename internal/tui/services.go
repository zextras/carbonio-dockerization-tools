package tui

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"carbonio-docker-cli/internal/config"
	"carbonio-docker-cli/internal/graph"
	"carbonio-docker-cli/internal/parser"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
				Foreground(lipgloss.Color("#666666")) // Grigio per nome required

	requiredTagStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")) // Bianco per tag required

	lockedTagStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")) // Grigio per tag locked

	inputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4"))
)

// ServicesConfirmedMsg is sent when user confirms selections
type ServicesConfirmedMsg struct {
	Backend  map[string]string // service_name: tag
	Frontend map[string]string // ui_name: tag (or "disabled")
}

// ServiceItem represents a selectable service
type ServiceItem struct {
	Name         string
	DefaultTag   string
	CustomTag    string
	Selected     bool
	IsBackend    bool
	IsRequired   bool // Se true, non può essere deselezionato
	TagLocked    bool // Se true, il tag non può essere modificato (es: local builds)
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

	viewOffset int // Scroll offset
	viewHeight int // Visible items count
}

// NewServicesModel creates a new services model
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

	// Create backend items - escludi SOLO i registrator (anche quelli local)
	var requiredBackend []*ServiceItem
	var optionalBackend []*ServiceItem

	backendNames := make([]string, 0, len(parsedConfig.BackendServices))
	for name := range parsedConfig.BackendServices {
		backendNames = append(backendNames, name)
	}
	sort.Strings(backendNames)

	for _, name := range backendNames {
		svc := parsedConfig.BackendServices[name]

		// NASCONDI i registrator dalla UI (sia normali che local)
		if svc.IsRegistrator {
			log.Printf("Hiding registrator from UI: %s (will be auto-added based on parent service)", name)
			continue
		}

		// NASCONDI carbonio-composed-ui (sarà sempre incluso automaticamente)
		if name == "carbonio-composed-ui" {
			log.Printf("Hiding carbonio-composed-ui from UI (always required, always local)")
			continue
		}

		// Ora mostriamo anche i servizi con tag "local", ma lockiamo il tag
		tagLocked := svc.DefaultTag == "local"

		// Usa DisplayName invece di Name per la visualizzazione
		displayName := svc.DisplayName
		if displayName == "" {
			displayName = name // Fallback al nome del servizio
		}

		item := &ServiceItem{
			Name:         displayName, // USA IL DISPLAY NAME
			DefaultTag:   svc.DefaultTag,
			CustomTag:    "",
			Selected:     true,
			IsBackend:    true,
			IsRequired:   svc.IsRequired,
			TagLocked:    tagLocked,
			Dependencies: svc.DependsOn,
		}

		// Separa required da optional
		if item.IsRequired {
			requiredBackend = append(requiredBackend, item)
		} else {
			optionalBackend = append(optionalBackend, item)
		}

		log.Printf("Backend item: %s (display=%s, tag=%s, required=%v, locked=%v)", name, displayName, svc.DefaultTag, svc.IsRequired, tagLocked)
	}

	// Combina: required prima, poi optional
	m.backendItems = append(requiredBackend, optionalBackend...)

	// Create frontend items - includi anche servizi con tag "local"
	var requiredFrontend []*ServiceItem
	var optionalFrontend []*ServiceItem

	frontendNames := make([]string, 0, len(parsedConfig.FrontendImages))
	for name := range parsedConfig.FrontendImages {
		frontendNames = append(frontendNames, name)
	}
	sort.Strings(frontendNames)

	for _, name := range frontendNames {
		ui := parsedConfig.FrontendImages[name]

		// Mostriamo anche le UI con tag "local", ma lockiamo il tag
		tagLocked := ui.DefaultTag == "local"

		item := &ServiceItem{
			Name:       name,
			DefaultTag: ui.DefaultTag,
			CustomTag:  "",
			Selected:   true,
			IsBackend:  false,
			IsRequired: ui.IsProxy,
			TagLocked:  tagLocked,
		}

		// Separa required da optional
		if item.IsRequired {
			requiredFrontend = append(requiredFrontend, item)
		} else {
			optionalFrontend = append(optionalFrontend, item)
		}

		log.Printf("Frontend item: %s (tag=%s, proxy=%v, locked=%v)", name, ui.DefaultTag, ui.IsProxy, tagLocked)
	}

	// Combina: required prima, poi optional
	m.frontendItems = append(requiredFrontend, optionalFrontend...)

	log.Printf("Created model with %d backend items, %d frontend items (registrators hidden, auto-managed)", len(m.backendItems), len(m.frontendItems))

	return m
}

func (m *ServicesModel) Init() tea.Cmd {
	// WindowSizeMsg will be sent automatically by bubbletea
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
			// Vai all'inizio
			m.cursor = 0
			m.viewOffset = 0
			log.Printf("Jump to start: cursor=0, offset=0")

		case "end", "G":
			// Vai alla fine
			totalItems := len(m.backendItems) + len(m.frontendItems)
			m.cursor = totalItems - 1
			m.adjustViewOffset()
			log.Printf("Jump to end: cursor=%d", m.cursor)

		case " ":
			// Toggle selection (solo se non required)
			m.toggleSelection()

		case "e":
			// Edit tag (solo se non locked)
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

	// HEADER FISSO - sempre in cima
	s.WriteString(titleStyle.Render("🔧 Select Services and UI Images"))
	s.WriteString("\n")

	// Formatta edition name in modo carino
	editionName := "CE (Community Edition)"
	if m.edition == parser.EditionAdvanced {
		editionName = "Advanced"
	}
	s.WriteString(helpStyle.Render(fmt.Sprintf("Edition: %s", editionName)))
	s.WriteString("\n\n")

	totalItems := len(m.backendItems) + len(m.frontendItems)

	// Calculate visible range
	startIdx := m.viewOffset
	endIdx := m.viewOffset + m.viewHeight
	if endIdx > totalItems {
		endIdx = totalItems
	}

	// SEMPRE mostra il titolo Services
	s.WriteString(sectionStyle.Render("Services:"))
	s.WriteString("\n")

	// Renderizza backend items visibili
	for i := 0; i < len(m.backendItems); i++ {
		if i >= startIdx && i < endIdx {
			s.WriteString(m.renderItem(i, m.backendItems[i]))
		}
	}

	// SEMPRE mostra il titolo Composed UI
	s.WriteString("\n")
	s.WriteString(sectionStyle.Render("Composed UI:"))
	s.WriteString("\n")

	// Renderizza frontend items visibili
	for i := 0; i < len(m.frontendItems); i++ {
		globalIndex := len(m.backendItems) + i
		if globalIndex >= startIdx && globalIndex < endIdx {
			s.WriteString(m.renderItem(globalIndex, m.frontendItems[i]))
		}
	}

	// FOOTER FISSO
	s.WriteString("\n")
	if m.editingTag {
		s.WriteString(helpStyle.Render("Editing tag: type to modify • enter: confirm • esc: cancel"))
	} else {
		s.WriteString(helpStyle.Render("↑/↓: navigate • g/G: top/bottom • space: toggle • e: edit tag • enter: start • x: export • q: quit"))
	}

	// Show position indicator
	s.WriteString("\n")
	s.WriteString(helpStyle.Render(fmt.Sprintf("Item %d/%d", m.cursor+1, totalItems)))

	return s.String()
}

func (m *ServicesModel) renderItem(index int, item *ServiceItem) string {
	cursor := " "
	if m.cursor == index {
		cursor = ">"
	}

	// Checkbox
	checkbox := "[ ]"
	if item.Selected {
		checkbox = "[✓]"
	}

	// Se required, mostra con simbolo diverso
	if item.IsRequired {
		checkbox = "[●]" // Sempre selezionato, non modificabile
	}

	// Tag display - NON mostrare "disabled" per UI deselezionate
	tag := item.DefaultTag
	if item.CustomTag != "" {
		tag = item.CustomTag
	}

	// Costruisci la linea in base allo stato
	var line string
	if item.IsRequired {
		// Required: nome grigio, tag bianco o grigio se locked
		namePart := requiredNameStyle.Render(fmt.Sprintf("%s %s %s", cursor, checkbox, item.Name))
		var tagPart string
		if item.TagLocked {
			tagPart = lockedTagStyle.Render(fmt.Sprintf("(%s)", tag))
		} else {
			tagPart = requiredTagStyle.Render(fmt.Sprintf("(%s)", tag))
		}
		line = namePart + " " + tagPart
	} else {
		// Normale: tutto stesso colore, ma tag locked in grigio
		if item.TagLocked {
			namePart := fmt.Sprintf("%s %s %s", cursor, checkbox, item.Name)
			tagPart := lockedTagStyle.Render(fmt.Sprintf("(%s)", tag))
			line = namePart + " " + tagPart
		} else {
			line = fmt.Sprintf("%s %s %s (%s)", cursor, checkbox, item.Name, tag)
		}
	}

	// Se stiamo editando questo item
	if m.editingTag && m.editingIndex == index {
		return selectedStyle.Render(line) + " ← " + inputStyle.Render(m.editingBuffer) + "\n"
	}

	// Stile in base allo stato
	if m.cursor == index {
		return selectedStyle.Render(line) + "\n"
	}

	return checkboxStyle.Render(line) + "\n"
}

func (m *ServicesModel) toggleSelection() {
	item := m.getCurrentItem()
	if item == nil || item.IsRequired {
		return // Non possiamo modificare i servizi required
	}

	item.Selected = !item.Selected
	log.Printf("Toggled service %s: selected=%v", item.Name, item.Selected)

	// Se backend service, handle dependencies
	if item.IsBackend && item.Selected {
		m.autoSelectDependencies(item)
	}
}

func (m *ServicesModel) autoSelectDependencies(item *ServiceItem) {
	// Get all dependencies
	deps := m.resolver.ResolveDependencies(item.Name)

	log.Printf("Auto-selecting dependencies for %s: %v", item.Name, deps)

	// Auto-select all dependencies
	for _, depName := range deps {
		for _, backendItem := range m.backendItems {
			if backendItem.Name == depName {
				backendItem.Selected = true
				log.Printf("  - Selected dependency: %s", depName)
			}
		}
	}
}

func (m *ServicesModel) startTagEdit() {
	item := m.getCurrentItem()
	if item == nil || item.TagLocked {
		// Non permettere editing se il tag è locked (es: local builds)
		log.Printf("Cannot edit tag for %s: tag is locked", item.Name)
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
			if item != nil && !item.TagLocked {
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
	log.Println("=== Confirm called ===")

	backend := make(map[string]string)
	frontend := make(map[string]string)

	log.Printf("Building backend selection from %d visible items", len(m.backendItems))

	// Backend: include servizi selezionati + loro registrator
	for _, item := range m.backendItems {
		if item.Selected {
			// Trova il servizio reale dal DisplayName
			var realServiceName string
			for name, svc := range m.parsedConfig.BackendServices {
				if svc.DisplayName == item.Name || name == item.Name {
					realServiceName = name
					break
				}
			}

			if realServiceName == "" {
				log.Printf("Warning: could not find real service name for display name %s", item.Name)
				continue
			}

			tag := item.DefaultTag
			if item.CustomTag != "" && !item.TagLocked {
				tag = item.CustomTag
			}
			backend[realServiceName] = tag
			log.Printf("  Backend: %s -> %s", realServiceName, tag)

			// Auto-includi TUTTI i registrator per questo servizio (usando la mappa)
			registrators := parser.GetRegistratorsForService(realServiceName)
			for _, regName := range registrators {
				if regSvc, exists := m.parsedConfig.BackendServices[regName]; exists {
					backend[regName] = regSvc.DefaultTag
					log.Printf("  Auto-added registrator: %s -> %s (for %s)", regName, regSvc.DefaultTag, realServiceName)
				}
			}
		}
	}

	// SEMPRE includi carbonio-composed-ui
	if composedUI, exists := m.parsedConfig.BackendServices["carbonio-composed-ui"]; exists {
		backend["carbonio-composed-ui"] = composedUI.DefaultTag
		log.Printf("  Auto-added carbonio-composed-ui: local")
	}

	log.Printf("Building frontend selection from %d visible items", len(m.frontendItems))

	// Frontend: TUTTI devono essere presenti
	for _, item := range m.frontendItems {
		tag := item.DefaultTag
		if item.CustomTag != "" && !item.TagLocked {
			tag = item.CustomTag
		}

		// Se non selezionato, usa "disabled" per docker ma NON nel display
		if !item.Selected {
			tag = "disabled"
		}

		frontend[item.Name] = tag
		log.Printf("  Frontend: %s -> %s", item.Name, tag)
	}

	log.Printf("Sending confirmation message with %d backend, %d frontend", len(backend), len(frontend))

	return m, func() tea.Msg {
		return ServicesConfirmedMsg{
			Backend:  backend,
			Frontend: frontend,
		}
	}
}

func (m *ServicesModel) exportConfig() (tea.Model, tea.Cmd) {
	backend := make(map[string]string)
	frontend := make(map[string]string)

	// Backend: servizi selezionati + loro registrator
	for _, item := range m.backendItems {
		if item.Selected {
			// Trova il servizio reale dal DisplayName
			var realServiceName string
			for name, svc := range m.parsedConfig.BackendServices {
				if svc.DisplayName == item.Name || name == item.Name {
					realServiceName = name
					break
				}
			}

			if realServiceName == "" {
				continue
			}

			tag := item.DefaultTag
			if item.CustomTag != "" && !item.TagLocked {
				tag = item.CustomTag
			}
			backend[realServiceName] = tag

			// Auto-includi tutti i registrator per questo servizio
			registrators := parser.GetRegistratorsForService(realServiceName)
			for _, regName := range registrators {
				if regSvc, exists := m.parsedConfig.BackendServices[regName]; exists {
					backend[regName] = regSvc.DefaultTag
				}
			}
		}
	}

	// Aggiungi carbonio-composed-ui
	if composedUI, exists := m.parsedConfig.BackendServices["carbonio-composed-ui"]; exists {
		backend["carbonio-composed-ui"] = composedUI.DefaultTag
	}

	// Frontend: TUTTI devono essere presenti
	for _, item := range m.frontendItems {
		tag := item.DefaultTag
		if item.CustomTag != "" && !item.TagLocked {
			tag = item.CustomTag
		}

		if !item.Selected {
			tag = "disabled"
		}

		frontend[item.Name] = tag
	}

	// Create config
	userConfig := config.CreateUserConfig(string(m.edition), backend, frontend)

	// Export to file
	filename := "carbonio-config.yaml"
	if err := config.ExportConfig(filename, userConfig); err != nil {
		// TODO: Show error
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
		m.viewHeight = 20 // Default if not set
	}

	totalItems := len(m.backendItems) + len(m.frontendItems)

	// Ensure cursor is within bounds
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= totalItems {
		m.cursor = totalItems - 1
	}

	// Adjust viewOffset to keep cursor visible
	if m.cursor < m.viewOffset {
		m.viewOffset = m.cursor
	}
	if m.cursor >= m.viewOffset+m.viewHeight {
		m.viewOffset = m.cursor - m.viewHeight + 1
	}

	// Ensure viewOffset is within bounds
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
