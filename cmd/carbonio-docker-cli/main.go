package main

import (
	"carbonio-docker-cli/internal/docker"
	"carbonio-docker-cli/internal/embedded"
	"carbonio-docker-cli/internal/tui"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// Global reference to the docker compose process for signal handling
var dockerComposeCmd *exec.Cmd

const registryHost = "registry.dev.zextras.com:443"
const minDockerComposeVersion = "2.30.0"

func main() {
	var configFile string
	var saveLogs bool
	var headless bool
	for i, arg := range os.Args[1:] {
		switch arg {
		case "--config-file":
			if i+1 < len(os.Args[1:]) {
				configFile = os.Args[i+2]
			}
		case "--save-logs":
			saveLogs = true
		case "--headless":
			headless = true
		}
	}
	if saveLogs {
		logFile, err := os.OpenFile("carbonio-docker-cli.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			fmt.Printf("Warning: Failed to open log file: %v\n", err)
		} else {
			defer logFile.Close()
			log.SetOutput(logFile)
			log.SetFlags(log.LstdFlags | log.Lshortfile)
		}
	} else {
		log.SetOutput(io.Discard)
	}
	log.Println("=== Starting Carbonio Docker CLI ===")
	log.Printf("Version: %s, Commit: %s, Date: %s", version, commit, date)
	log.Printf("Config file: %s", configFile)
	log.Printf("Save logs: %v", saveLogs)

	fmt.Println("🔍 Checking Docker Compose version...")
	if err := checkDockerComposeVersion(); err != nil {
		fmt.Printf("\n❌ Error: %v\n", err)
		fmt.Printf("   Please upgrade Docker Compose to version %s or higher.\n", minDockerComposeVersion)
		fmt.Println("   See: https://docs.docker.com/compose/install/")
		log.Printf("Docker Compose version check failed: %v", err)
		os.Exit(1)
	}
	fmt.Println("✓ Docker Compose version is compatible\n")
	log.Println("Docker Compose version check passed")

	fmt.Println("🔍 Checking Docker Engine version...")
	needsConfirmation, err := checkDockerEngineVersion()
	if err != nil {
		log.Printf("Docker Engine version check warning: %v", err)
	}
	if needsConfirmation && !headless {
		fmt.Println()
		fmt.Print("Press Enter to continue or Ctrl+C to quit...")
		var input string
		fmt.Scanln(&input)
		fmt.Println()
	}
	log.Println("Docker Engine version check completed")

	fmt.Println("🔍 Checking registry connectivity...")
	if err := checkRegistryConnectivity(); err != nil {
		fmt.Printf("\n❌ Error: Registry unavailable (%s)\n", registryHost)
		fmt.Println("   Please check your VPN connection and try again.")
		log.Printf("Registry check failed: %v", err)
		os.Exit(1)
	}
	fmt.Println("✓ Registry is reachable\n")
	log.Println("Registry connectivity check passed")

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

	workDir := extractor.GetWorkDir()
	setupSignalHandler(workDir)

	if headless && configFile != "" {
		log.Println("Starting in headless mode...")
		if err := runHeadless(workDir, configFile); err != nil {
			log.Fatalf("Headless mode error: %v", err)
		}
	} else {
		log.Println("Starting TUI application...")
		app := tui.NewApp(workDir, configFile)
		if err := app.Run(); err != nil {
			log.Fatalf("Application error: %v", err)
		}
	}
	log.Println("=== Application finished ===")
}

func checkDockerComposeVersion() error {
	cmd := exec.Command("docker", "compose", "version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("docker compose not found or not working: %w", err)
	}

	versionStr := string(output)
	log.Printf("Docker Compose version output: %s", versionStr)

	re := regexp.MustCompile(`v?(\d+)\.(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(versionStr)
	if len(matches) < 4 {
		return fmt.Errorf("unable to parse docker compose version from: %s", versionStr)
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])

	currentVersion := fmt.Sprintf("%d.%d.%d", major, minor, patch)
	log.Printf("Detected Docker Compose version: %s", currentVersion)

	minParts := strings.Split(minDockerComposeVersion, ".")
	minMajor, _ := strconv.Atoi(minParts[0])
	minMinor, _ := strconv.Atoi(minParts[1])
	minPatch, _ := strconv.Atoi(minParts[2])

	if major < minMajor {
		return fmt.Errorf("docker compose version %s is too old (minimum required: %s)", currentVersion, minDockerComposeVersion)
	}
	if major == minMajor && minor < minMinor {
		return fmt.Errorf("docker compose version %s is too old (minimum required: %s)", currentVersion, minDockerComposeVersion)
	}
	if major == minMajor && minor == minMinor && patch < minPatch {
		return fmt.Errorf("docker compose version %s is too old (minimum required: %s)", currentVersion, minDockerComposeVersion)
	}

	return nil
}

func checkDockerEngineVersion() (bool, error) {
	cmd := exec.Command("docker", "version", "--format", "{{.Server.Version}}")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to get docker engine version: %w", err)
	}

	versionStr := strings.TrimSpace(string(output))
	log.Printf("Docker Engine version output: %s", versionStr)

	re := regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(versionStr)
	if len(matches) < 4 {
		return false, fmt.Errorf("unable to parse docker engine version from: %s", versionStr)
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])

	currentVersion := fmt.Sprintf("%d.%d.%d", major, minor, patch)
	log.Printf("Detected Docker Engine version: %s", currentVersion)

	if major >= 29 {
		fmt.Println("⚠️  Warning: Docker Engine 29+ detected")
		fmt.Println("   Traefik v3 has a known incompatibility with Docker 29+ that causes API errors.")
		fmt.Println("   The routing will still work via Consul, but you'll see error logs from Traefik.")
		fmt.Println("")
		fmt.Println("   Temporary fix - Configure Docker daemon:")
		fmt.Println("   1. Edit /etc/docker/daemon.json:")
		fmt.Println("      sudo nano /etc/docker/daemon.json")
		fmt.Println("")
		fmt.Println("   2. Add this configuration:")
		fmt.Println("      {")
		fmt.Println("        \"min-api-version\": \"1.24\"")
		fmt.Println("      }")
		fmt.Println("")
		fmt.Println("   3. Restart Docker:")
		fmt.Println("      sudo systemctl restart docker")
		fmt.Println("")
		fmt.Println("   Alternative: Downgrade to Docker Engine 28.x")
		fmt.Println("   Issue: https://github.com/traefik/traefik/issues/12253")
		fmt.Println("")
		log.Printf("Docker 29+ detected - displayed Traefik compatibility warning")
		return true, nil
	}

	return false, nil
}

func checkRegistryConnectivity() error {
	timeout := 5 * time.Second
	conn, err := net.DialTimeout("tcp", registryHost, timeout)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer conn.Close()
	return nil
}

func setupSignalHandler(workDir string) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Received interrupt signal, cleaning up...")
		fmt.Println("\n🧹 Cleaning up containers...")

		// Kill docker compose subprocess if running
		if dockerComposeCmd != nil && dockerComposeCmd.Process != nil {
			log.Println("Killing docker compose subprocess...")
			// Send SIGTERM to the process group
			syscall.Kill(-dockerComposeCmd.Process.Pid, syscall.SIGTERM)
			// Wait a moment for graceful shutdown
			time.Sleep(2 * time.Second)
			// Force kill if still running
			dockerComposeCmd.Process.Kill()
		}

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

func runHeadless(workDir, configFile string) error {
	log.Printf("=== Running headless with config: %s ===", configFile)

	executor := docker.NewExecutor(workDir)

	// Cleanup existing containers first
	fmt.Println("🧹 Cleaning up existing containers...")
	if err := executor.CleanupAll(); err != nil {
		log.Printf("Warning: initial cleanup failed: %v", err)
	}
	fmt.Println("✓ Cleanup complete\n")

	// Use CommandBuilder to get proper env vars and command
	fmt.Println("📋 Loading configuration...")

	editionStr, err := loadConfigEdition(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config edition: %w", err)
	}

	// Import parser and config packages
	edition := "ce"
	if editionStr == "advanced" {
		edition = "advanced"
	}

	fmt.Printf("   Edition: %s\n", edition)
	fmt.Println("🚀 Starting Docker Compose in headless mode...")
	fmt.Println("   Press Ctrl+C to stop and cleanup\n")

	// Build the docker compose command with env vars from config
	args := []string{"compose", "-f", "docker-compose.yaml"}
	if edition == "advanced" {
		args = append(args, "-f", "docker-compose-advanced.yaml")
	}
	args = append(args, "up")

	// Read config and build env vars
	envVars := buildEnvVarsFromConfig(configFile)
	log.Printf("Environment variables: %s", envVars)

	// Run docker compose with env vars
	cmd := exec.Command("docker", args...)
	cmd.Dir = workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// Set process group so we can kill the entire group
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Parse and add env vars
	env := os.Environ()
	for _, v := range strings.Fields(envVars) {
		env = append(env, v)
	}
	cmd.Env = env

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start docker compose: %w", err)
	}

	// Store global reference for signal handler
	dockerComposeCmd = cmd

	log.Printf("Docker Compose started with PID %d", cmd.Process.Pid)

	// Wait for the process (it will be interrupted by signal handler)
	err = cmd.Wait()
	if err != nil {
		log.Printf("Docker compose exited: %v", err)
	}

	return nil
}

func buildEnvVarsFromConfig(configFile string) string {
	data, err := os.ReadFile(configFile)
	if err != nil {
		log.Printf("Failed to read config: %v", err)
		return ""
	}

	content := string(data)
	var envPairs []string

	// Parse backend services
	lines := strings.Split(content, "\n")
	inBackend := false
	currentService := ""

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "backend:" {
			inBackend = true
			continue
		}
		if trimmed == "frontend:" {
			inBackend = false
			continue
		}

		// Skip non-service lines
		if strings.HasPrefix(trimmed, "#") || trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "version:") || strings.HasPrefix(trimmed, "edition:") || strings.HasPrefix(trimmed, "carbonio:") {
			continue
		}

		// Service name (ends with :)
		if strings.HasSuffix(trimmed, ":") && !strings.HasPrefix(trimmed, "image:") && !strings.HasPrefix(trimmed, "tag:") {
			currentService = strings.TrimSuffix(trimmed, ":")
			continue
		}

		// Image line
		if strings.HasPrefix(trimmed, "image:") && currentService != "" {
			image := strings.TrimSpace(strings.TrimPrefix(trimmed, "image:"))
			envName := serviceToEnvName(currentService, inBackend)
			if envName != "" {
				envPairs = append(envPairs, fmt.Sprintf("%s_IMAGE=%s", envName, image))
			}
		}

		// Tag line
		if strings.HasPrefix(trimmed, "tag:") && currentService != "" {
			tag := strings.TrimSpace(strings.TrimPrefix(trimmed, "tag:"))
			envName := serviceToEnvName(currentService, inBackend)
			if envName != "" {
				envPairs = append(envPairs, fmt.Sprintf("%s_TAG=%s", envName, tag))
			}
		}
	}

	return strings.Join(envPairs, " ")
}

func serviceToEnvName(service string, isBackend bool) string {
	// Convert service name to env var name
	// e.g., carbonio-ws-collaboration -> CARBONIO_WS_COLLABORATION
	name := strings.ToUpper(strings.ReplaceAll(service, "-", "_"))
	return name
}

func loadConfigEdition(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	content := string(data)
	if strings.Contains(content, "edition: advanced") {
		return "advanced", nil
	}
	return "ce", nil
}
