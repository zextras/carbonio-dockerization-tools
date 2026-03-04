package gui

import (
	"carbonio-dockerization-tools/internal/config"
	"carbonio-dockerization-tools/internal/docker"
	"carbonio-dockerization-tools/internal/parser"
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
	window           fyne.Window
	workDir          string
	logPath          string
	appVersion       string
	parsedConfig     *parser.ParsedConfig
	userConfig       *config.UserConfig
	edition          parser.Edition
	executor         *docker.Executor
	pendingBackend   map[string]*config.ImageConfig
	pendingFrontend  map[string]*config.ImageConfig
	cleanPersistence bool
}

func NewApp(workDir string, logPath string, appVersion string, window fyne.Window) *App {
	a := &App{
		window:     window,
		workDir:    workDir,
		logPath:    logPath,
		appVersion: appVersion,
		executor:   docker.NewExecutor(workDir),
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
			showErrorDialog(err.Error(), a.window)
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
			showErrorDialog(fmt.Sprintf("failed to open log file: %v", err), a.window)
			return
		}
		defer src.Close()

		dst, err := os.Create(destPath)
		if err != nil {
			showErrorDialog(fmt.Sprintf("failed to create export file: %v", err), a.window)
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, src); err != nil {
			showErrorDialog(fmt.Sprintf("failed to copy logs: %v", err), a.window)
			return
		}

		showSuccessDialog("Logs Exported", fmt.Sprintf("Saved to:\n%s", destPath), a.window)
	}, a.window)
	fd.Show()
}

func (a *App) loadConfigFromFile(filePath string) error {
	editionStr, err := config.LoadConfigEdition(filePath)
	if err != nil {
		return fmt.Errorf("failed to read config edition: %v", err)
	}

	edition := parser.EditionCE
	if editionStr == "advanced" {
		edition = parser.EditionAdvanced
	}

	parsedConfig, err := parser.ParseAll(a.workDir, edition)
	if err != nil {
		return fmt.Errorf("failed to parse docker files: %v", err)
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

func (a *App) buildDockerCommand() (*docker.BuildResult, error) {
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
		a.appVersion,
	)
	return config.ExportConfig(filePath, userConfig)
}
