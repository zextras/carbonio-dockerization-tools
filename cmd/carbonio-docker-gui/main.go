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

	// Show loading screen immediately and run preflight checks
	runPreflightChecks(w, logPath)

	w.ShowAndRun()
}

func runPreflightChecks(w fyne.Window, logPath string) {
	updateStep := gui.ShowLoadingScreen(w)

	go func() {
		// Check Docker Compose
		fyne.Do(func() { updateStep("Checking Docker Compose...") })
		if err := preflight.CheckDockerComposeVersion(); err != nil {
			fyne.Do(func() {
				gui.ShowFatalErrorDialog(fmt.Sprintf("Docker Compose check failed: %v\n\nPlease upgrade to version %s or higher.\nSee: https://docs.docker.com/compose/install/", err, preflight.MinDockerComposeVersion), w)
			})
			return
		}

		// Check registry connectivity
		fyne.Do(func() { updateStep("Checking registry connectivity...") })
		if err := preflight.CheckRegistryConnectivity(); err != nil {
			fyne.Do(func() {
				gui.ShowVPNRetryDialog(
					fmt.Sprintf("Registry unavailable (%s)\n\nPlease check your VPN connection and try again.", preflight.RegistryHost),
					w,
					func() { runPreflightChecks(w, logPath) },
				)
			})
			return
		}

		fyne.Do(func() {
			continueStartup(w, logPath, updateStep)
		})
	}()
}

func continueStartup(w fyne.Window, logPath string, updateStep func(string)) {
	updateStep("Extracting files...")

	go func() {
		extractor, err := embedded.NewExtractor()
		if err != nil {
			fyne.Do(func() {
				gui.ShowFatalErrorDialog(fmt.Sprintf("Failed to initialize: %v", err), w)
			})
			return
		}
		if err := extractor.EnsureExtracted(); err != nil {
			fyne.Do(func() {
				gui.ShowFatalErrorDialog(fmt.Sprintf("Failed to extract files: %v", err), w)
			})
			return
		}

		workDir := extractor.GetWorkDir()
		guiApp := gui.NewApp(workDir, logPath, w)

		fyne.Do(func() { updateStep("Cleaning up previous sessions...") })
		guiApp.RunInitialCleanup()

		fyne.Do(func() {
			guiApp.ShowStartupScreen()
		})
	}()
}
