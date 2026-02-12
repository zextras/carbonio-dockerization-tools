package main

import (
	"carbonio-docker-cli/internal/embedded"
	"carbonio-docker-cli/internal/gui"
	"carbonio-docker-cli/internal/preflight"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Setup logging
	logDir := filepath.Join(os.Getenv("HOME"), ".local", "state", "carbonio-dockerization")
	os.MkdirAll(logDir, 0755)
	logPath := filepath.Join(logDir, "gui.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to open log file: %v\n", err)
	} else {
		defer logFile.Close()
		log.SetOutput(logFile)
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	log.Println("=== Starting Carbonio Docker GUI ===")
	log.Printf("Version: %s, Commit: %s, Date: %s", version, commit, date)

	a := app.NewWithID("com.zextras.carbonio-dockerization-gui")
	a.Settings().SetTheme(gui.NewCarbonioTheme())
	a.SetIcon(gui.LogoResource())
	w := a.NewWindow("Carbonio dockerization GUI")
	w.SetIcon(gui.LogoResource())
	w.Resize(fyne.NewSize(900, 700))

	// Preflight checks
	if err := preflight.CheckDockerComposeVersion(); err != nil {
		dialog.ShowError(fmt.Errorf("Docker Compose check failed: %w\n\nPlease upgrade to version %s or higher.\nSee: https://docs.docker.com/compose/install/", err, preflight.MinDockerComposeVersion), w)
		w.ShowAndRun()
		return
	}

	if err := preflight.CheckRegistryConnectivity(); err != nil {
		dialog.ShowError(fmt.Errorf("Registry unavailable (%s)\n\nPlease check your VPN connection and try again.", preflight.RegistryHost), w)
		w.ShowAndRun()
		return
	}

	continueStartup(w, a, logPath)
	w.ShowAndRun()
}

func continueStartup(w fyne.Window, a fyne.App, logPath string) {
	extractor, err := embedded.NewExtractor()
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to initialize: %v", err), w)
		return
	}
	if err := extractor.EnsureExtracted(); err != nil {
		dialog.ShowError(fmt.Errorf("Failed to extract files: %v", err), w)
		return
	}

	workDir := extractor.GetWorkDir()
	guiApp := gui.NewApp(workDir, logPath, w)

	// Initial cleanup
	log.Println("Running initial cleanup...")
	guiApp.RunInitialCleanup()

	guiApp.ShowStartupScreen()
}
