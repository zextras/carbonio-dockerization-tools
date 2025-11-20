package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"carbonio-docker-cli/internal/docker"
	"carbonio-docker-cli/internal/embedded"
	"carbonio-docker-cli/internal/tui"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

const registryHost = "registry.dev.zextras.com:443"

func main() {
	// Parse flags
	var configFile string
	var saveLogs bool

	for i, arg := range os.Args[1:] {
		switch arg {
		case "--config-file":
			if i+1 < len(os.Args[1:]) {
				configFile = os.Args[i+2]
			}
		case "--save-logs":
			saveLogs = true
		}
	}

	// Setup logging - SOLO se richiesto
	if saveLogs {
		logFile, err := os.OpenFile("carbonio-docker-cli.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0755)
		if err != nil {
			fmt.Printf("Warning: Failed to open log file: %v\n", err)
		} else {
			defer logFile.Close()
			log.SetOutput(logFile)
			log.SetFlags(log.LstdFlags | log.Lshortfile)
		}
	} else {
		// Disabilita logging
		log.SetOutput(io.Discard)
	}

	log.Println("=== Starting Carbonio Docker CLI ===")
	log.Printf("Version: %s, Commit: %s, Date: %s", version, commit, date)
	log.Printf("Config file: %s", configFile)
	log.Printf("Save logs: %v", saveLogs)

	// Check registry connectivity
	fmt.Println("🔍 Checking registry connectivity...")
	if err := checkRegistryConnectivity(); err != nil {
		fmt.Printf("\n❌ Error: Registry unavailable (%s)\n", registryHost)
		fmt.Println("   Please check your VPN connection and try again.")
		log.Printf("Registry check failed: %v", err)
		os.Exit(1)
	}
	fmt.Println("✓ Registry is reachable\n")
	log.Println("Registry connectivity check passed")

	// Extract embedded files
	fmt.Println("🔧 Preparing Carbonio environment...")
	extractor, err := embedded.NewExtractor()
	if err != nil {
		log.Fatalf("Failed to initialize extractor: %v", err)
	}

	log.Println("Extracting embedded files...")
	if err := extractor.EnsureExtracted(); err != nil {
		log.Fatalf("Failed to extract files: %v", err)
	}
	fmt.Println("✓ Environment ready\n")
	log.Printf("Working directory: %s", extractor.GetWorkDir())

	// Setup signal handler for Ctrl+C - cleanup before exit
	workDir := extractor.GetWorkDir()
	setupSignalHandler(workDir)

	// Start TUI application
	log.Println("Starting TUI application...")
	app := tui.NewApp(workDir, configFile)
	if err := app.Run(); err != nil {
		log.Fatalf("Application error: %v", err)
	}

	log.Println("=== Application finished ===")
}

// checkRegistryConnectivity verifies if the Docker registry is reachable
func checkRegistryConnectivity() error {
	timeout := 5 * time.Second

	conn, err := net.DialTimeout("tcp", registryHost, timeout)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer conn.Close()

	return nil
}

// setupSignalHandler configura un handler per Ctrl+C che fa cleanup
func setupSignalHandler(workDir string) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received interrupt signal, cleaning up...")
		fmt.Println("\n🧹 Cleaning up containers...")

		executor := docker.NewExecutor(workDir)
		if err := executor.CleanupAll(); err != nil {
			log.Printf("Cleanup failed: %v", err)
			fmt.Printf("Warning: Cleanup failed: %v\n", err)
		} else {
			fmt.Println("✓ Cleanup complete")
			log.Println("Cleanup successful")
		}

		os.Exit(0)
	}()
}
