// SPDX-FileCopyrightText: 2026 Zextras <https://www.zextras.com>
//
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"carbonio-dockerization-tools/internal/embedded"
	"carbonio-dockerization-tools/internal/gui"
	"carbonio-dockerization-tools/internal/logutil"
	"carbonio-dockerization-tools/internal/preflight"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	ensurePATH()

	// Setup logging
	logDir := filepath.Join(os.Getenv("HOME"), ".local", "state", "carbonio-dockerization")
	os.MkdirAll(logDir, 0755)
	logPath := filepath.Join(logDir, "gui.log")
	logutil.TrimLogFile(logPath)
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
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

// ensurePATH adds common Docker install locations to PATH on macOS.
// GUI .app bundles inherit a minimal PATH (/usr/bin:/bin:/usr/sbin:/sbin)
// that doesn't include directories where Docker Desktop places its binaries.
func ensurePATH() {
	if runtime.GOOS != "darwin" {
		return
	}
	extra := []string{
		"/usr/local/bin",
		"/opt/homebrew/bin",
		"/Applications/Docker.app/Contents/Resources/bin",
	}
	current := os.Getenv("PATH")
	for _, p := range extra {
		if !strings.Contains(current, p) {
			current += ":" + p
		}
	}
	os.Setenv("PATH", current)
}

func runPreflightChecks(w fyne.Window, logPath string) {
	updateStep := gui.ShowLoadingScreen(w)

	go func() {
		// Check Docker Compose
		fyne.Do(func() { updateStep("Checking Docker Compose...") })
		if err := preflight.CheckDockerComposeVersion(); err != nil {
			fyne.Do(func() {
				gui.ShowFatalErrorDialog(fmt.Sprintf("Docker Compose check failed: %v\n\nPlease upgrade to version %s or higher.\nSee: https://docs.docker.com/compose/install/", err, preflight.MinDockerComposeVersion), w, func() { gui.ShowGuideDialog(w) }, logPath)
			})
			return
		}

		// Check Docker daemon is running
		fyne.Do(func() { updateStep("Checking Docker daemon...") })
		if err := preflight.CheckDockerDaemon(); err != nil {
			fyne.Do(func() {
				gui.ShowVPNRetryDialog(
					fmt.Sprintf("%v", err),
					w,
					func() { runPreflightChecks(w, logPath) },
					func() { gui.ShowGuideDialog(w) },
				)
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
					func() { gui.ShowGuideDialog(w) },
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
				gui.ShowFatalErrorDialog(fmt.Sprintf("Failed to initialize: %v", err), w, nil, logPath)
			})
			return
		}
		if err := extractor.EnsureExtracted(); err != nil {
			fyne.Do(func() {
				gui.ShowFatalErrorDialog(fmt.Sprintf("Failed to extract files: %v", err), w, nil, logPath)
			})
			return
		}

		workDir := extractor.GetWorkDir()
		natIP, err := preflight.DetectNATIP()
		if err != nil {
			log.Printf("Warning: failed to detect NAT IP: %v", err)
		}

		guiApp := gui.NewApp(workDir, logPath, version, natIP, w)

		fyne.Do(func() { updateStep("Cleaning up previous sessions...") })
		guiApp.RunInitialCleanup()

		// Register global close intercept: always clean up on window close
		registerGlobalCloseIntercept(w, guiApp)

		// Register OS signal handler for cleanup on SIGTERM/SIGINT
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigChan
			log.Println("Signal received, cleaning up before exit...")
			guiApp.CleanupAllEditions()
			os.Exit(0)
		}()

		fyne.Do(func() {
			guiApp.SetupMainMenu()
			guiApp.ShowStartupScreen()
			guiApp.CheckForUpdate()
		})
	}()
}

// registerGlobalCloseIntercept sets a safety-net close intercept on the window
// that ensures all Docker containers are cleaned up when the GUI is closed,
// regardless of which screen is active. This prevents leftover containers
// from holding ports after the app exits.
func registerGlobalCloseIntercept(w fyne.Window, guiApp *gui.App) {
	w.SetCloseIntercept(func() {
		log.Println("Window close intercepted, running global cleanup...")
		go func() {
			guiApp.CleanupAllEditions()
			fyne.Do(func() {
				w.SetCloseIntercept(nil)
				w.Close()
			})
		}()
	})
}
