package main

import (
	"carbonio-docker-cli/internal/embedded"
	"carbonio-docker-cli/internal/tui"
	"fmt"
	"log"
	"os"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Parse flags
	var logToFile bool
	var configFile string

	for i, arg := range os.Args[1:] {
		switch arg {
		case "--version", "-v":
			fmt.Printf("carbonio-docker-cli %s\n", version)
			fmt.Printf("  commit: %s\n", commit)
			fmt.Printf("  built:  %s\n", date)
			os.Exit(0)
		case "--log-to-file":
			logToFile = true
		case "--config":
			if i+1 < len(os.Args[1:]) {
				configFile = os.Args[i+2]
			}
		case "--help", "-h":
			printHelp()
			os.Exit(0)
		}
	}

	// Setup logging
	if logToFile {
		logFile, err := os.OpenFile("carbonio-docker-cli.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Fatalf("Failed to open log file: %v", err)
		}
		defer logFile.Close()
		log.SetOutput(logFile)
	}

	// Extract embedded files
	fmt.Println("🔧 Preparing Carbonio environment...")
	extractor, err := embedded.NewExtractor()
	if err != nil {
		log.Fatalf("Failed to initialize extractor: %v", err)
	}

	if err := extractor.EnsureExtracted(); err != nil {
		log.Fatalf("Failed to extract files: %v", err)
	}
	fmt.Println("✓ Environment ready\n")

	// Start TUI application
	app := tui.NewApp(extractor.GetWorkDir(), configFile)
	if err := app.Run(); err != nil {
		log.Fatalf("Application error: %v", err)
	}
}

func printHelp() {
	fmt.Println("Carbonio Docker CLI")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  carbonio-docker-cli [flags]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --version, -v       Show version information")
	fmt.Println("  --log-to-file       Save logs to carbonio-docker-cli.log")
	fmt.Println("  --config <file>     Import configuration from YAML file")
	fmt.Println("  --help, -h          Show this help message")
}
