package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"time"

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

	for i, arg := range os.Args[1:] {
		switch arg {
		case "--version", "-v":
			fmt.Printf("carbonio-docker-cli %s\n", version)
			fmt.Printf("  commit: %s\n", commit)
			fmt.Printf("  built:  %s\n", date)
			os.Exit(0)
		case "--config":
			if i+1 < len(os.Args[1:]) {
				configFile = os.Args[i+2]
			}
		case "--help", "-h":
			printHelp()
			os.Exit(0)
		}
	}

	// Setup logging - SEMPRE attivo per debug
	logFile, err := os.OpenFile("carbonio-docker-cli.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Printf("Warning: Failed to open log file: %v\n", err)
	} else {
		defer logFile.Close()
		log.SetOutput(logFile)
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	log.Println("=== Starting Carbonio Docker CLI ===")
	log.Printf("Version: %s, Commit: %s, Date: %s", version, commit, date)
	log.Printf("Config file: %s", configFile)

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

	// Start TUI application
	log.Println("Starting TUI application...")
	app := tui.NewApp(extractor.GetWorkDir(), configFile)
	if err := app.Run(); err != nil {
		log.Fatalf("Application error: %v", err)
	}

	log.Println("=== Application finished ===")
}

func printHelp() {
	fmt.Println("Carbonio Docker CLI")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  carbonio-docker-cli [flags]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --version, -v       Show version information")
	fmt.Println("  --config <file>     Import configuration from YAML file")
	fmt.Println("  --help, -h          Show this help message")
	fmt.Println()
	fmt.Println("Logs are always saved to: carbonio-docker-cli.log")
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
