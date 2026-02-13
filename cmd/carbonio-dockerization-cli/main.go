package main

import (
	"carbonio-dockerization-tools/internal/config"
	"carbonio-dockerization-tools/internal/docker"
	"carbonio-dockerization-tools/internal/embedded"
	"carbonio-dockerization-tools/internal/parser"
	"carbonio-dockerization-tools/internal/preflight"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	var configFile string
	var clean bool

	for i, arg := range os.Args[1:] {
		switch arg {
		case "--config-file":
			if i+1 < len(os.Args[1:]) {
				configFile = os.Args[i+2]
			}
		case "--clean":
			clean = true
		}
	}

	if configFile == "" {
		fmt.Fprintln(os.Stderr, "Error: --config-file is required")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  carbonio-dockerization-cli --config-file <file> [--clean]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "For interactive use, run 'Carbonio dockerization GUI' instead.")
		os.Exit(1)
	}

	// Setup logging
	logDir := filepath.Join(os.Getenv("HOME"), ".local", "state", "carbonio-dockerization")
	os.MkdirAll(logDir, 0755)
	logFile, err := os.OpenFile(filepath.Join(logDir, "cli.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to open log file: %v\n", err)
	} else {
		defer logFile.Close()
		log.SetOutput(logFile)
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	log.Println("=== Starting Carbonio Docker CLI ===")
	log.Printf("Version: %s, Commit: %s, Date: %s", version, commit, date)
	log.Printf("Config file: %s", configFile)

	// Preflight checks
	fmt.Fprintln(os.Stderr, "Checking Docker Compose version...")
	if err := preflight.CheckDockerComposeVersion(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Please upgrade Docker Compose to version %s or higher.\n", preflight.MinDockerComposeVersion)
		fmt.Fprintln(os.Stderr, "See: https://docs.docker.com/compose/install/")
		os.Exit(1)
	}

	fmt.Fprintln(os.Stderr, "Checking registry connectivity...")
	if err := preflight.CheckRegistryConnectivity(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Registry unavailable (%s)\n", preflight.RegistryHost)
		fmt.Fprintln(os.Stderr, "Please check your VPN connection and try again.")
		os.Exit(1)
	}

	fmt.Fprintln(os.Stderr, "Preparing Carbonio environment...")
	extractor, err := embedded.NewExtractor()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to initialize extractor: %v\n", err)
		os.Exit(1)
	}
	if err := extractor.EnsureExtracted(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to extract files: %v\n", err)
		os.Exit(1)
	}

	workDir := extractor.GetWorkDir()
	executor := docker.NewExecutor(workDir)

	fmt.Fprintln(os.Stderr, "Cleaning up existing containers...")
	if err := executor.CleanupAll(); err != nil {
		log.Printf("Warning: cleanup failed: %v", err)
	}

	// Load and validate config
	editionStr, err := config.LoadConfigEdition(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	edition := parser.EditionCE
	if editionStr == "advanced" {
		edition = parser.EditionAdvanced
	}
	executor.SetEdition(editionStr)

	parsedConfig, err := parser.ParseAll(workDir, edition)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to parse docker files: %v\n", err)
		os.Exit(1)
	}
	userConfig, err := config.LoadConfig(configFile, parsedConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Force clean if edition changed, otherwise clean only if requested
	editionChanged := executor.HasEditionChanged()
	if editionChanged {
		fmt.Fprintln(os.Stderr, "Edition changed since last run, cleaning all persistence...")
	}
	if clean || editionChanged {
		if err := executor.CleanAllVolumes(); err != nil {
			log.Printf("Warning: volume cleanup failed: %v", err)
		}
		fmt.Println()
	}
	executor.SaveLastEdition()

	// Build and execute docker compose
	builder := docker.NewCommandBuilder(workDir, edition, parsedConfig)
	for serviceName, imgConfig := range userConfig.Carbonio.Backend {
		builder.SetBackendService(serviceName, imgConfig)
	}
	for uiName, imgConfig := range userConfig.Carbonio.Frontend {
		builder.SetFrontendImage(uiName, imgConfig)
	}
	envVars, cmdParts, err := builder.Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to build command: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintln(os.Stderr, "Starting Carbonio services...")
	fmt.Fprintln(os.Stderr, "Press Ctrl+C to stop and cleanup")
	fmt.Println()

	outputChan := make(chan string, 100)

	go func() {
		err := executor.ExecuteWithSignalHandler(envVars, cmdParts, outputChan, true)
		if err != nil {
			log.Printf("Docker execution error: %v", err)
		}
	}()

	for line := range outputChan {
		fmt.Println(line)
	}

	fmt.Println()
	fmt.Println("Docker Compose process has exited.")
	fmt.Fprintln(os.Stderr, "Running cleanup...")
	if err := executor.CleanupAll(); err != nil {
		log.Printf("Warning: cleanup failed: %v", err)
	}
	fmt.Fprintln(os.Stderr, "Cleanup complete")
}
