package tui

import (
	"fmt"
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zextras/carbonio-base-dockerization/cli/internal/config"
	"github.com/zextras/carbonio-base-dockerization/cli/internal/docker"
	"github.com/zextras/carbonio-base-dockerization/cli/internal/graph"
	"github.com/zextras/carbonio-base-dockerization/cli/internal/parser"
)

// Screen represents different screens in the application
type Screen int

const (
	ScreenStartup Screen = iota
	ScreenEdition
	ScreenServices
	ScreenMonitor
)

// App is the main TUI application
type App struct {
	workDir       string
	configFile    string
	currentScreen Screen

	// Models for each screen
	startupModel  *StartupModel
	editionModel  *EditionModel
	servicesModel *ServicesModel
	monitorModel  *MonitorModel

	// Shared data
	parsedConfig *parser.ParsedConfig
	userConfig   *config.UserConfig
	edition      parser.Edition
	executor     *docker.Executor
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
	// Cleanup any existing containers first
	fmt.Println("🧹 Cleaning up existing containers...")
	if err := a.executor.CleanupExisting("advanced"); err != nil {
		log.Printf("Warning: cleanup failed: %v", err)
	}
	fmt.Println("✓ Cleanup complete\n")

	// If config file provided, load it directly
	if a.configFile != "" {
		return a.runWithConfig()
	}

	// Start interactive mode
	a.startupModel = NewStartupModel()

	p := tea.NewProgram(a, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}

// runWithConfig loads config and starts directly
func (a *App) runWithConfig() error {
	// Parse docker files first
	parsedConfig, err := parser.ParseAll(a.workDir, parser.EditionCE)
	if err != nil {
		return fmt.Errorf("failed to parse docker files: %w", err)
	}

	// Load user config
	userConfig, err := config.LoadConfig(a.configFile, parsedConfig)
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

	// Build and execute command
	return a.executeFromConfig()
}

// executeFromConfig executes docker compose from loaded config
func (a *App) executeFromConfig() error {
	builder := docker.NewCommandBuilder(a.workDir, a.edition, a.parsedConfig)

	// Set backend services
	for serviceName, tag := range a.userConfig.Carbonio.Backend {
		builder.SetBackendService(serviceName, tag)
	}

	// Set frontend images
	for uiName, tag := range a.userConfig.Carbonio.Frontend {
		builder.SetFrontendImage(uiName, tag)
	}

	// Build command
	envVars, cmdParts, err := builder.Build()
	if err != nil {
		return fmt.Errorf("failed to build command: %w", err)
	}

	fmt.Printf("🚀 Starting Carbonio %s...\n\n", a.edition)
	fmt.Printf("Command: %s %s\n\n", envVars, cmdParts)

	// Execute
	outputChan := make(chan string, 100)

	go func() {
		for line := range outputChan {
			fmt.Println(line)
		}
	}()

	return a.executor.Execute(envVars, cmdParts, outputChan)
}

// Bubbletea Model interface implementation

func (a *App) Init() tea.Cmd {
	return nil
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}

	case StartupChoiceMsg:
		if msg.ImportConfig {
			// User wants to import config - show file picker
			return a.handleImportConfig()
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
		// User confirmed service selection - execute
		return a.handleServicesConfirmed(msg)

	case MonitorCompletedMsg:
		// Docker compose finished
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
	case ScreenMonitor:
		if a.monitorModel != nil {
			return a.monitorModel.View()
		}
	}

	return "Loading..."
}

// Helper methods

func (a *App) handleImportConfig() (tea.Model, tea.Cmd) {
	// TODO: Implement file picker
	// For now, just show error
	fmt.Println("File picker not implemented yet. Use --config flag instead.")
	return a, tea.Quit
}

func (a *App) handleEditionChoice() (tea.Model, tea.Cmd) {
	// Parse docker files with selected edition
	parsedConfig, err := parser.ParseAll(a.workDir, a.edition)
	if err != nil {
		log.Printf("Failed to parse docker files: %v", err)
		return a, tea.Quit
	}

	a.parsedConfig = parsedConfig
	a.currentScreen = ScreenServices

	// Create dependency resolver
	resolver := graph.NewDependencyResolver(parsedConfig.BackendServices)

	a.servicesModel = NewServicesModel(parsedConfig, resolver, a.edition)
	return a, nil
}

func (a *App) handleServicesConfirmed(msg ServicesConfirmedMsg) (tea.Model, tea.Cmd) {
	// Build docker compose command
	builder := docker.NewCommandBuilder(a.workDir, a.edition, a.parsedConfig)

	// Set backend services
	for serviceName, tag := range msg.Backend {
		builder.SetBackendService(serviceName, tag)
	}

	// Set frontend images
	for uiName, tag := range msg.Frontend {
		builder.SetFrontendImage(uiName, tag)
	}

	// Build command
	envVars, cmdParts, err := builder.Build()
	if err != nil {
		log.Printf("Failed to build command: %v", err)
		return a, tea.Quit
	}

	// Switch to monitor screen
	a.currentScreen = ScreenMonitor
	a.monitorModel = NewMonitorModel(a.executor, envVars, cmdParts)

	return a, a.monitorModel.Start()
}
