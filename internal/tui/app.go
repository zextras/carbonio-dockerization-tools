package tui

import (
	"fmt"
	"log"

	"carbonio-docker-cli/internal/config"
	"carbonio-docker-cli/internal/docker"
	"carbonio-docker-cli/internal/graph"
	"carbonio-docker-cli/internal/parser"
	tea "github.com/charmbracelet/bubbletea"
)

// Screen represents different screens in the application
type Screen int

const (
	ScreenStartup Screen = iota
	ScreenEdition
	ScreenServices
	ScreenConfigSave
	ScreenFilePicker
	ScreenMonitor
)

// App is the main TUI application
type App struct {
	workDir       string
	configFile    string
	currentScreen Screen

	// Models for each screen
	startupModel    *StartupModel
	editionModel    *EditionModel
	servicesModel   *ServicesModel
	configSaveModel *ConfigSaveModel
	filePickerModel *FilePickerModel
	monitorModel    *MonitorModel

	// Shared data
	parsedConfig    *parser.ParsedConfig
	userConfig      *config.UserConfig
	edition         parser.Edition
	executor        *docker.Executor
	pendingBackend  map[string]string
	pendingFrontend map[string]string
}

// NewApp creates a new application instance
func NewApp(workDir, configFile string) *App {
	return &App{
		workDir:       workDir,
		configFile:    configFile,
		currentScreen: ScreenStartup,
		executor:      docker.NewExecutor(workDir),
	}
}

// Run starts the TUI application
func (a *App) Run() error {
	// Cleanup any existing containers first (CE + Advanced)
	fmt.Println("🧹 Cleaning up existing containers...")
	if err := a.executor.CleanupAll(); err != nil {
		log.Printf("Warning: cleanup failed: %v", err)
	}
	fmt.Println("✓ Cleanup complete\n")

	// If config file provided via flag, load it and go directly to monitor
	if a.configFile != "" {
		return a.runWithConfig(a.configFile)
	}

	// Start interactive mode
	a.startupModel = NewStartupModel()

	p := tea.NewProgram(a, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}

// runWithConfig loads config from file and starts directly with monitor screen
func (a *App) runWithConfig(filePath string) error {
	log.Printf("=== Running with config file: %s ===", filePath)

	// Parse docker files first (with CE to get basic structure)
	parsedConfig, err := parser.ParseAll(a.workDir, parser.EditionCE)
	if err != nil {
		return fmt.Errorf("failed to parse docker files: %w", err)
	}

	// Load user config
	userConfig, err := config.LoadConfig(filePath, parsedConfig)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Set edition
	edition := parser.EditionCE
	if userConfig.Carbonio.Edition == "advanced" {
		edition = parser.EditionAdvanced
	}

	// Re-parse with correct edition
	parsedConfig, err = parser.ParseAll(a.workDir, edition)
	if err != nil {
		return fmt.Errorf("failed to parse docker files: %w", err)
	}

	a.parsedConfig = parsedConfig
	a.userConfig = userConfig
	a.edition = edition
	a.pendingBackend = userConfig.Carbonio.Backend
	a.pendingFrontend = userConfig.Carbonio.Frontend

	// Build docker compose command
	log.Println("Building docker command from config...")
	builder := docker.NewCommandBuilder(a.workDir, a.edition, a.parsedConfig)

	// Set backend services
	for serviceName, tag := range a.pendingBackend {
		builder.SetBackendService(serviceName, tag)
		log.Printf("Backend: %s -> %s", serviceName, tag)
	}

	// Set frontend images
	for uiName, tag := range a.pendingFrontend {
		builder.SetFrontendImage(uiName, tag)
		log.Printf("Frontend: %s -> %s", uiName, tag)
	}

	// Build command
	envVars, cmdParts, err := builder.Build()
	if err != nil {
		return fmt.Errorf("failed to build command: %w", err)
	}

	log.Printf("Command built successfully")

	// Build visible services list from pending data
	visibleServices := a.buildVisibleServicesList(a.pendingBackend)

	// Start TUI with monitor screen directly
	a.currentScreen = ScreenMonitor
	a.monitorModel = NewMonitorModel(a.executor, envVars, cmdParts, visibleServices)

	p := tea.NewProgram(a, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}

// buildVisibleServicesList creates a list of services to show in monitor
// Excludes only hidden services (registrators, provisioner, etc.)
func (a *App) buildVisibleServicesList(backendServices map[string]string) []string {
	visibleServices := []string{}

	// Add all non-hidden backend services
	for serviceName := range backendServices {
		if !parser.GlobalDockerConfig.IsServiceHidden(serviceName) {
			visibleServices = append(visibleServices, serviceName)
		}
	}

	return visibleServices
}

// Bubbletea Model interface implementation

func (a *App) Init() tea.Cmd {
	// If we're starting directly in monitor screen (from config file)
	if a.currentScreen == ScreenMonitor && a.monitorModel != nil {
		return a.monitorModel.Start()
	}
	return nil
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" && a.currentScreen != ScreenMonitor {
			// Allow ctrl+c to quit in non-monitor screens
			// Monitor screen handles its own cleanup
			return a, tea.Quit
		}

	case StartupChoiceMsg:
		if msg.ImportConfig {
			// User wants to import config - show file picker
			a.currentScreen = ScreenFilePicker
			a.filePickerModel = NewFilePickerModel()
			return a, a.filePickerModel.Init()
		} else {
			// User wants custom config - show edition selection
			a.currentScreen = ScreenEdition
			a.editionModel = NewEditionModel()
			return a, nil
		}

	case EditionChoiceMsg:
		// User selected edition - parse configs and show services
		a.edition = msg.Edition
		return a.handleEditionChoice()

	case ServicesConfirmedMsg:
		// User confirmed service selection - go to config save screen
		a.pendingBackend = msg.Backend
		a.pendingFrontend = msg.Frontend
		a.currentScreen = ScreenConfigSave
		a.configSaveModel = NewConfigSaveModel(a.edition)
		return a, nil

	case ConfigSaveChoiceMsg:
		// User decided about config save
		if msg.WantsSave && msg.Filename != "" {
			// Save config
			if err := a.saveConfig(msg.Filename); err != nil {
				log.Printf("Failed to save config: %v", err)
				// Continue anyway
			}
		}
		// Now execute
		return a.handleExecute()

	case FilePickerChoiceMsg:
		// User selected a config file
		if msg.Cancelled {
			// Go back to startup
			a.currentScreen = ScreenStartup
			a.startupModel = NewStartupModel()
			return a, nil
		}

		// Load and execute config
		return a.handleFilePickerChoice(msg.FilePath)

	case MonitorCompletedMsg:
		// Docker compose finished
		log.Println("Docker compose finished in app")
		return a, tea.Quit
	}

	// Delegate to current screen
	switch a.currentScreen {
	case ScreenStartup:
		if a.startupModel != nil {
			newModel, cmd := a.startupModel.Update(msg)
			a.startupModel = newModel.(*StartupModel)
			return a, cmd
		}

	case ScreenEdition:
		if a.editionModel != nil {
			newModel, cmd := a.editionModel.Update(msg)
			a.editionModel = newModel.(*EditionModel)
			return a, cmd
		}

	case ScreenServices:
		if a.servicesModel != nil {
			newModel, cmd := a.servicesModel.Update(msg)
			a.servicesModel = newModel.(*ServicesModel)
			return a, cmd
		}

	case ScreenConfigSave:
		if a.configSaveModel != nil {
			newModel, cmd := a.configSaveModel.Update(msg)
			a.configSaveModel = newModel.(*ConfigSaveModel)
			return a, cmd
		}

	case ScreenFilePicker:
		if a.filePickerModel != nil {
			newModel, cmd := a.filePickerModel.Update(msg)
			a.filePickerModel = newModel.(*FilePickerModel)
			return a, cmd
		}

	case ScreenMonitor:
		if a.monitorModel != nil {
			newModel, cmd := a.monitorModel.Update(msg)
			a.monitorModel = newModel.(*MonitorModel)
			return a, cmd
		}
	}

	return a, nil
}

func (a *App) View() string {
	switch a.currentScreen {
	case ScreenStartup:
		if a.startupModel != nil {
			return a.startupModel.View()
		}
	case ScreenEdition:
		if a.editionModel != nil {
			return a.editionModel.View()
		}
	case ScreenServices:
		if a.servicesModel != nil {
			return a.servicesModel.View()
		}
	case ScreenConfigSave:
		if a.configSaveModel != nil {
			return a.configSaveModel.View()
		}
	case ScreenFilePicker:
		if a.filePickerModel != nil {
			return a.filePickerModel.View()
		}
	case ScreenMonitor:
		if a.monitorModel != nil {
			return a.monitorModel.View()
		}
	}

	return "Loading..."
}

// Helper methods

func (a *App) handleEditionChoice() (tea.Model, tea.Cmd) {
	log.Printf("=== Edition choice: %s ===", a.edition)

	// Parse docker files with selected edition
	log.Println("Parsing docker files...")
	parsedConfig, err := parser.ParseAll(a.workDir, a.edition)
	if err != nil {
		log.Printf("ERROR: Failed to parse docker files: %v", err)
		return a, tea.Quit
	}

	log.Printf("Parsed successfully: %d backend, %d frontend",
		len(parsedConfig.BackendServices),
		len(parsedConfig.FrontendImages))

	a.parsedConfig = parsedConfig
	a.currentScreen = ScreenServices

	// Create dependency resolver
	resolver := graph.NewDependencyResolver(parsedConfig.BackendServices)

	log.Println("Creating services model...")
	a.servicesModel = NewServicesModel(parsedConfig, resolver, a.edition)

	return a, nil
}

func (a *App) handleExecute() (tea.Model, tea.Cmd) {
	log.Println("=== Executing services ===")
	log.Printf("Backend services: %d", len(a.pendingBackend))
	log.Printf("Frontend images: %d", len(a.pendingFrontend))

	// Build docker compose command
	builder := docker.NewCommandBuilder(a.workDir, a.edition, a.parsedConfig)

	// Set backend services
	log.Println("Setting backend services...")
	for serviceName, tag := range a.pendingBackend {
		builder.SetBackendService(serviceName, tag)
		log.Printf("  - %s: %s", serviceName, tag)
	}

	// Set frontend images
	log.Println("Setting frontend images...")
	for uiName, tag := range a.pendingFrontend {
		builder.SetFrontendImage(uiName, tag)
		log.Printf("  - %s: %s", uiName, tag)
	}

	// Build command
	log.Println("Building docker command...")
	envVars, cmdParts, err := builder.Build()
	if err != nil {
		log.Printf("ERROR: Failed to build command: %v", err)
		return a, tea.Quit
	}

	log.Printf("Command built successfully")
	log.Printf("Env vars: %s", envVars)
	log.Printf("Cmd parts: %v", cmdParts)

	// Build visible services list
	visibleServices := a.buildVisibleServicesList(a.pendingBackend)

	log.Printf("Visible services for monitoring: %v", visibleServices)

	// Switch to monitor screen
	a.currentScreen = ScreenMonitor
	a.monitorModel = NewMonitorModel(a.executor, envVars, cmdParts, visibleServices)

	log.Println("Starting monitor screen")
	return a, a.monitorModel.Start()
}

func (a *App) handleFilePickerChoice(filePath string) (tea.Model, tea.Cmd) {
	log.Printf("=== File picker choice: %s ===", filePath)

	// Parse docker files first
	parsedConfig, err := parser.ParseAll(a.workDir, parser.EditionCE)
	if err != nil {
		log.Printf("ERROR: Failed to parse docker files: %v", err)
		fmt.Printf("\n❌ Error parsing docker files: %v\n", err)
		return a, tea.Quit
	}

	// Load user config
	userConfig, err := config.LoadConfig(filePath, parsedConfig)
	if err != nil {
		log.Printf("ERROR: Failed to load config: %v", err)
		fmt.Printf("\n❌ Error loading config: %v\n", err)
		return a, tea.Quit
	}

	// Set edition
	edition := parser.EditionCE
	if userConfig.Carbonio.Edition == "advanced" {
		edition = parser.EditionAdvanced
	}

	// Re-parse with correct edition
	parsedConfig, err = parser.ParseAll(a.workDir, edition)
	if err != nil {
		log.Printf("ERROR: Failed to parse docker files: %v", err)
		fmt.Printf("\n❌ Error parsing docker files: %v\n", err)
		return a, tea.Quit
	}

	a.parsedConfig = parsedConfig
	a.userConfig = userConfig
	a.edition = edition
	a.pendingBackend = userConfig.Carbonio.Backend
	a.pendingFrontend = userConfig.Carbonio.Frontend

	// Go directly to execute
	return a.handleExecute()
}

func (a *App) saveConfig(filename string) error {
	log.Printf("Saving config to: %s", filename)

	// Create user config
	userConfig := config.CreateUserConfig(
		string(a.edition),
		a.pendingBackend,
		a.pendingFrontend,
	)

	// Export to file
	if err := config.ExportConfig(filename, userConfig); err != nil {
		return fmt.Errorf("failed to export config: %w", err)
	}

	log.Printf("Config saved successfully to: %s", filename)
	return nil
}
