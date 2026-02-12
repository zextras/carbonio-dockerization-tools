package gui

import (
	"carbonio-docker-cli/internal/config"
	"carbonio-docker-cli/internal/docker"
	"carbonio-docker-cli/internal/parser"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

type App struct {
	window          fyne.Window
	workDir         string
	logPath         string
	parsedConfig    *parser.ParsedConfig
	userConfig      *config.UserConfig
	edition         parser.Edition
	executor        *docker.Executor
	pendingBackend  map[string]*config.ImageConfig
	pendingFrontend map[string]*config.ImageConfig
	cleanDatabase   bool
}

func NewApp(workDir string, logPath string, window fyne.Window) *App {
	a := &App{
		window:   window,
		workDir:  workDir,
		logPath:  logPath,
		executor: docker.NewExecutor(workDir),
	}
	a.setupMainMenu()
	return a
}

func (a *App) setupMainMenu() {
	exportLogs := fyne.NewMenuItem("Export Logs...", a.exportLogs)
	toolsMenu := fyne.NewMenu("Tools", exportLogs)
	a.window.SetMainMenu(fyne.NewMainMenu(toolsMenu))
}

func (a *App) exportLogs() {
	fd := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil {
			dialog.ShowError(err, a.window)
			return
		}
		if uri == nil {
			return
		}

		timestamp := time.Now().Format("2006-01-02_150405")
		destName := fmt.Sprintf("carbonio-dockerization-gui-logs-%s.log", timestamp)
		destPath := filepath.Join(uri.Path(), destName)

		src, err := os.Open(a.logPath)
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to open log file: %w", err), a.window)
			return
		}
		defer src.Close()

		dst, err := os.Create(destPath)
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to create export file: %w", err), a.window)
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, src); err != nil {
			dialog.ShowError(fmt.Errorf("failed to copy logs: %w", err), a.window)
			return
		}

		dialog.ShowInformation("Logs Exported", fmt.Sprintf("Saved to:\n%s", destPath), a.window)
	}, a.window)
	fd.Show()
}

func (a *App) loadConfigFromFile(filePath string) error {
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
		return err
	}

	a.parsedConfig = parsedConfig
	a.userConfig = userConfig
	a.edition = edition
	a.executor.SetEdition(editionStr)
	a.pendingBackend = userConfig.Carbonio.Backend
	a.pendingFrontend = userConfig.Carbonio.Frontend

	return nil
}

func (a *App) buildDockerCommand() (string, []string, error) {
	builder := docker.NewCommandBuilder(a.workDir, a.edition, a.parsedConfig)
	for serviceName, imgConfig := range a.pendingBackend {
		builder.SetBackendService(serviceName, imgConfig)
	}
	for uiName, imgConfig := range a.pendingFrontend {
		builder.SetFrontendImage(uiName, imgConfig)
	}
	return builder.Build()
}

func (a *App) RunInitialCleanup() {
	log.Println("Running initial cleanup...")
	if err := a.executor.CleanupAll(); err != nil {
		log.Printf("Warning: cleanup failed: %v", err)
	}
}

func (a *App) saveConfig(filePath string) error {
	userConfig := config.CreateUserConfig(
		string(a.edition),
		a.pendingBackend,
		a.pendingFrontend,
	)
	return config.ExportConfig(filePath, userConfig)
}
