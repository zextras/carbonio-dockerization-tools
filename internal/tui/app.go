package tui

import (
	"carbonio-docker-cli/internal/config"
	"carbonio-docker-cli/internal/docker"
	"carbonio-docker-cli/internal/graph"
	"carbonio-docker-cli/internal/parser"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"log"
)

type Screen int

const (
	ScreenStartup Screen = iota
	ScreenEdition
	ScreenServices
	ScreenConfigSave
	ScreenFilePicker
	ScreenMonitor
)

type App struct {
	workDir         string
	configFile      string
	headless        bool
	currentScreen   Screen
	startupModel    *StartupModel
	editionModel    *EditionModel
	servicesModel   *ServicesModel
	configSaveModel *ConfigSaveModel
	filePickerModel *FilePickerModel
	monitorModel    *MonitorModel
	parsedConfig    *parser.ParsedConfig
	userConfig      *config.UserConfig
	edition         parser.Edition
	executor        *docker.Executor
	pendingBackend  map[string]*config.ImageConfig
	pendingFrontend map[string]*config.ImageConfig
}

func NewApp(workDir, configFile string, headless bool) *App {
	return &App{
		workDir:       workDir,
		configFile:    configFile,
		headless:      headless,
		currentScreen: ScreenStartup,
		executor:      docker.NewExecutor(workDir),
	}
}
func (a *App) Run() error {
	fmt.Println("🧹 Cleaning up existing containers and pruning system...")
	if err := a.executor.CleanupAll(); err != nil {
		log.Printf("Warning: cleanup failed: %v", err)
	}
	fmt.Println("✓ Cleanup complete\n")
	if a.configFile != "" {
		return a.runWithConfig(a.configFile)
	}
	a.startupModel = NewStartupModel()
	p := tea.NewProgram(a, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}
func (a *App) runWithConfig(filePath string) error {
	log.Printf("=== Running with config file: %s ===", filePath)
	editionStr, err := config.LoadConfigEdition(filePath)
	if err != nil {
		return fmt.Errorf("failed to read config edition: %w", err)
	}
	edition := parser.EditionCE
	if editionStr == "advanced" {
		edition = parser.EditionAdvanced
	}
	parsedConfig, err := parser.ParseAll(a.workDir, edition)
	if err != nil {
		return fmt.Errorf("failed to parse docker files: %w", err)
	}
	userConfig, err := config.LoadConfig(filePath, parsedConfig)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	a.parsedConfig = parsedConfig
	a.userConfig = userConfig
	a.edition = edition
	a.pendingBackend = userConfig.Carbonio.Backend
	a.pendingFrontend = userConfig.Carbonio.Frontend
	log.Println("Building docker command from config...")
	builder := docker.NewCommandBuilder(a.workDir, a.edition, a.parsedConfig)
	for serviceName, imgConfig := range a.pendingBackend {
		builder.SetBackendService(serviceName, imgConfig)
		log.Printf("Backend: %s -> %s:%s", serviceName, imgConfig.Image, imgConfig.Tag)
	}
	for uiName, imgConfig := range a.pendingFrontend {
		builder.SetFrontendImage(uiName, imgConfig)
		log.Printf("Frontend: %s -> %s:%s", uiName, imgConfig.Image, imgConfig.Tag)
	}
	envVars, cmdParts, err := builder.Build()
	if err != nil {
		return fmt.Errorf("failed to build command: %w", err)
	}
	log.Printf("Command built successfully")

	// Headless mode: run docker compose directly without TUI
	if a.headless {
		return a.runHeadless(envVars, cmdParts)
	}

	// Interactive mode: use TUI
	visibleServices := a.buildVisibleServicesList(a.pendingBackend)
	a.currentScreen = ScreenMonitor
	a.monitorModel = NewMonitorModel(a.executor, envVars, cmdParts, visibleServices, a.workDir)
	p := tea.NewProgram(a, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}

func (a *App) runHeadless(envVars string, cmdParts []string) error {
	fmt.Println("🚀 Starting Carbonio services in headless mode...")
	fmt.Println("   Press Ctrl+C to stop and cleanup")
	fmt.Println()

	outputChan := make(chan string, 100)

	// Start docker compose in a goroutine
	go func() {
		err := a.executor.Execute(envVars, cmdParts, outputChan)
		if err != nil {
			log.Printf("Docker execution error: %v", err)
		}
	}()

	// Read output and print to stdout
	for line := range outputChan {
		fmt.Println(line)
	}

	fmt.Println()
	fmt.Println("Docker Compose process has exited.")

	// Always run cleanup when exiting headless mode
	fmt.Println("🧹 Running cleanup...")
	if err := a.executor.CleanupAll(); err != nil {
		log.Printf("Warning: cleanup failed: %v", err)
	}
	fmt.Println("✓ Cleanup complete")

	return nil
}
func (a *App) buildVisibleServicesList(backendServices map[string]*config.ImageConfig) []string {
	visibleServices := []string{}
	for serviceName := range backendServices {
		if !parser.GlobalDockerConfig.IsServiceHidden(serviceName) {
			visibleServices = append(visibleServices, serviceName)
		}
	}
	return visibleServices
}
func (a *App) Init() tea.Cmd {
	if a.currentScreen == ScreenMonitor && a.monitorModel != nil {
		return a.monitorModel.Start()
	}
	return nil
}
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" && a.currentScreen != ScreenMonitor {
			return a, tea.Quit
		}
	case StartupChoiceMsg:
		if msg.ImportConfig {
			a.currentScreen = ScreenFilePicker
			a.filePickerModel = NewFilePickerModel()
			return a, a.filePickerModel.Init()
		} else {
			a.currentScreen = ScreenEdition
			a.editionModel = NewEditionModel()
			return a, nil
		}
	case EditionChoiceMsg:
		a.edition = msg.Edition
		return a.handleEditionChoice()
	case ServicesConfirmedMsg:
		a.pendingBackend = msg.Backend
		a.pendingFrontend = msg.Frontend
		a.currentScreen = ScreenConfigSave
		a.configSaveModel = NewConfigSaveModel(a.edition)
		return a, nil
	case ConfigSaveChoiceMsg:
		if msg.WantsSave && msg.Filename != "" {
			if err := a.saveConfig(msg.Filename); err != nil {
				log.Printf("Failed to save config: %v", err)
			}
		}
		return a.handleExecute()
	case FilePickerChoiceMsg:
		if msg.Cancelled {
			a.currentScreen = ScreenStartup
			a.startupModel = NewStartupModel()
			return a, nil
		}
		return a.handleFilePickerChoice(msg.FilePath)
	case MonitorCompletedMsg:
		log.Println("Docker compose finished in app")
		return a, tea.Quit
	}
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
func (a *App) handleEditionChoice() (tea.Model, tea.Cmd) {
	log.Printf("=== Edition choice: %s ===", a.edition)
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
	resolver := graph.NewDependencyResolver(parsedConfig.BackendServices)
	log.Println("Creating services model...")
	a.servicesModel = NewServicesModel(parsedConfig, resolver, a.edition)
	return a, nil
}
func (a *App) handleExecute() (tea.Model, tea.Cmd) {
	log.Println("=== Executing services ===")
	log.Printf("Backend services: %d", len(a.pendingBackend))
	log.Printf("Frontend images: %d", len(a.pendingFrontend))
	builder := docker.NewCommandBuilder(a.workDir, a.edition, a.parsedConfig)
	log.Println("Setting backend services...")
	for serviceName, imgConfig := range a.pendingBackend {
		builder.SetBackendService(serviceName, imgConfig)
		log.Printf("  - %s: %s:%s", serviceName, imgConfig.Image, imgConfig.Tag)
	}
	log.Println("Setting frontend images...")
	for uiName, imgConfig := range a.pendingFrontend {
		builder.SetFrontendImage(uiName, imgConfig)
		log.Printf("  - %s: %s:%s", uiName, imgConfig.Image, imgConfig.Tag)
	}
	log.Println("Building docker command...")
	envVars, cmdParts, err := builder.Build()
	if err != nil {
		log.Printf("ERROR: Failed to build command: %v", err)
		return a, tea.Quit
	}
	log.Printf("Command built successfully")
	log.Printf("Env vars: %s", envVars)
	log.Printf("Cmd parts: %v", cmdParts)
	visibleServices := a.buildVisibleServicesList(a.pendingBackend)
	log.Printf("Visible services for monitoring: %v", visibleServices)
	a.currentScreen = ScreenMonitor
	a.monitorModel = NewMonitorModel(a.executor, envVars, cmdParts, visibleServices, a.workDir)
	log.Println("Starting monitor screen")
	return a, a.monitorModel.Start()
}
func (a *App) handleFilePickerChoice(filePath string) (tea.Model, tea.Cmd) {
	log.Printf("=== File picker choice: %s ===", filePath)
	editionStr, err := config.LoadConfigEdition(filePath)
	if err != nil {
		log.Printf("ERROR: Failed to read config edition: %v", err)
		fmt.Printf("\n❌ Error reading config edition: %v\n", err)
		return a, tea.Quit
	}
	edition := parser.EditionCE
	if editionStr == "advanced" {
		edition = parser.EditionAdvanced
	}
	parsedConfig, err := parser.ParseAll(a.workDir, edition)
	if err != nil {
		log.Printf("ERROR: Failed to parse docker files: %v", err)
		fmt.Printf("\n❌ Error parsing docker files: %v\n", err)
		return a, tea.Quit
	}
	userConfig, err := config.LoadConfig(filePath, parsedConfig)
	if err != nil {
		log.Printf("ERROR: Failed to load config: %v", err)
		fmt.Printf("\n❌ Error loading config: %v\n", err)
		return a, tea.Quit
	}
	a.parsedConfig = parsedConfig
	a.userConfig = userConfig
	a.edition = edition
	a.pendingBackend = userConfig.Carbonio.Backend
	a.pendingFrontend = userConfig.Carbonio.Frontend
	return a.handleExecute()
}
func (a *App) saveConfig(filename string) error {
	log.Printf("Saving config to: %s", filename)
	userConfig := config.CreateUserConfig(
		string(a.edition),
		a.pendingBackend,
		a.pendingFrontend,
	)
	if err := config.ExportConfig(filename, userConfig); err != nil {
		return fmt.Errorf("failed to export config: %w", err)
	}
	log.Printf("Config saved successfully to: %s", filename)
	return nil
}
