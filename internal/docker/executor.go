package docker

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Executor struct {
	workDir     string
	cmd         *exec.Cmd
	cleanupOnce sync.Once
	cleanupErr  error
}

func NewExecutor(workDir string) *Executor {
	return &Executor{
		workDir: workDir,
	}
}

func (e *Executor) CleanupExisting(edition string) error {
	return e.CleanupAll()
}

func (e *Executor) CleanupAll() error {
	return e.cleanupAllWithOutput(true)
}

func (e *Executor) CleanupAllQuiet() error {
	return e.cleanupAllWithOutput(false)
}

func isNonLinuxPlatform() bool {
	return runtime.GOOS != "linux"
}

func getDockerEnv() []string {
	env := os.Environ()
	if isNonLinuxPlatform() {
		env = append(env, "DOCKER_DEFAULT_PLATFORM=linux/amd64")
		log.Printf("Non-Linux platform detected (%s), setting DOCKER_DEFAULT_PLATFORM=linux/amd64", runtime.GOOS)
	}
	return env
}

func (e *Executor) createDockerCommand(args ...string) *exec.Cmd {
	cmd := exec.Command("docker", args...)
	cmd.Dir = e.workDir
	cmd.Env = getDockerEnv()
	return cmd
}

func (e *Executor) cleanupAllWithOutput(showOutput bool) error {
	log.Println("Running complete cleanup (CE + Advanced)...")
	fmt.Printf("Working directory: %s\n", e.workDir)

	fmt.Println("Stopping containers gracefully...")
	stopCmd := e.createDockerCommand(
		"compose",
		"--project-name", "carbonio",
		"-f", "docker-compose.yaml",
		"-f", "docker-compose-advanced.yaml",
		"stop",
		"--timeout", "60",
	)
	stopCmd.Stdout = os.Stdout
	stopCmd.Stderr = os.Stderr
	if err := stopCmd.Run(); err != nil {
		fmt.Printf("Warning: compose stop failed: %v (continuing anyway)\n", err)
	} else {
		fmt.Println("Compose stop completed")
	}

	time.Sleep(2 * time.Second)

	maxRetries := 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		fmt.Printf("Cleanup attempt %d/%d...\n", attempt, maxRetries)

		cmd := e.createDockerCommand(
			"compose",
			"--project-name", "carbonio",
			"-f", "docker-compose.yaml",
			"-f", "docker-compose-advanced.yaml",
			"down",
			"--remove-orphans",
			"--timeout", "30",
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err == nil {
			log.Println("Compose down completed successfully")
			lastErr = nil
			break
		}

		errStr := err.Error()
		if strings.Contains(errStr, "is restarting") || strings.Contains(errStr, "wait until the container is running") {
			log.Printf("Container is restarting, waiting before retry %d/%d...", attempt, maxRetries)
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt*3) * time.Second)
				continue
			}
		}

		lastErr = err
		log.Printf("Compose down attempt %d failed: %v", attempt, err)

		if attempt < maxRetries {
			time.Sleep(time.Duration(attempt*2) * time.Second)
		}
	}

	if lastErr != nil {
		log.Printf("Warning: compose down failed after %d attempts: %v (continuing to prune)", maxRetries, lastErr)
	}

	log.Println("Removing consul data volume to prevent rejoin errors...")
	consulVolumeNames := []string{
		"carbonio_consul-data",
		"consul-data",
	}
	for _, volName := range consulVolumeNames {
		rmCmd := e.createDockerCommand("volume", "rm", "-f", volName)
		if err := rmCmd.Run(); err != nil {
			log.Printf("Volume %s removal: %v (may not exist, this is fine)", volName, err)
		} else {
			log.Printf("Volume %s removed successfully", volName)
		}
	}

	log.Println("Running docker system prune...")
	for attempt := 1; attempt <= 2; attempt++ {
		pruneCmd := e.createDockerCommand("system", "prune", "-f")
		if showOutput {
			pruneCmd.Stdout = os.Stdout
			pruneCmd.Stderr = os.Stderr
		}

		if err := pruneCmd.Run(); err != nil {
			log.Printf("System prune attempt %d failed: %v", attempt, err)
			if attempt < 2 {
				time.Sleep(2 * time.Second)
				continue
			}
			return fmt.Errorf("system prune failed: %w", err)
		}

		log.Println("System prune completed")
		break
	}

	log.Println("Cleanup completed")
	return nil
}

// CleanDatabaseVolumes removes PostgreSQL and storages volumes for a fresh installation
func (e *Executor) CleanDatabaseVolumes() error {
	fmt.Println("🗑️  Removing database and storage volumes for fresh installation...")

	// First, stop containers that use these volumes
	containersToStop := []string{
		"carbonio-carbonio-postgres-1",
		"carbonio-carbonio-storages-1",
		"carbonio-carbonio-files-1",
		"carbonio-carbonio-preview-1",
		"carbonio-carbonio-docs-editor-1",
		"carbonio-carbonio-docs-connector-1",
		"carbonio-carbonio-tasks-1",
	}

	fmt.Println("   Stopping containers that use database volumes...")
	for _, container := range containersToStop {
		stopCmd := e.createDockerCommand("stop", "-t", "5", container)
		stopCmd.Run() // Ignore errors - container may not exist
		rmCmd := e.createDockerCommand("rm", "-f", container)
		rmCmd.Run() // Ignore errors
	}

	// Give Docker a moment to release the volumes
	time.Sleep(2 * time.Second)

	// PostgreSQL and storages volume names (both naming conventions)
	volumeNames := []string{
		// PostgreSQL volumes (metadata)
		"carbonio-postgres-data",
		"carbonio-postgres-data-advanced",
		"carbonio_postgres-data",
		"carbonio_postgres-data-advanced",
		// Storages volumes (actual files)
		"carbonio-storages-data",
		"carbonio-storages-data-advanced",
		"carbonio_storages-data",
		"carbonio_storages-data-advanced",
	}

	removedCount := 0
	for _, volName := range volumeNames {
		rmCmd := e.createDockerCommand("volume", "rm", "-f", volName)
		if err := rmCmd.Run(); err != nil {
			log.Printf("Volume %s removal: %v (may not exist)", volName, err)
		} else {
			fmt.Printf("   Removed volume: %s\n", volName)
			removedCount++
		}
	}

	if removedCount > 0 {
		fmt.Printf("✓ Removed %d volume(s)\n", removedCount)
	} else {
		fmt.Println("✓ No volumes to remove")
	}

	return nil
}

func (e *Executor) Execute(envVars string, cmdParts []string, outputChan chan string) error {
	return e.ExecuteWithSignalHandler(envVars, cmdParts, outputChan, false)
}

func (e *Executor) ExecuteWithSignalHandler(envVars string, cmdParts []string, outputChan chan string, handleSignals bool) error {
	cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
	cmd.Dir = e.workDir

	env := getDockerEnv()
	env = append(env, parseEnvVars(envVars)...)
	cmd.Env = env

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start docker compose: %w", err)
	}

	e.cmd = cmd

	// Only register signal handler in headless mode
	// In TUI mode, Bubbletea handles Ctrl+C and calls StopAndCleanup
	if handleSignals {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		go func() {
			sig := <-sigChan
			log.Printf("!!! Signal received in executor: %v, killing docker compose process...", sig)
			// Just kill the process - cleanup will be done by runHeadless
			if e.cmd != nil && e.cmd.Process != nil {
				e.cmd.Process.Kill()
			}
		}()
	}

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			log.Println(line)
			outputChan <- line
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			log.Println(line)
			outputChan <- line
		}
	}()

	err = cmd.Wait()
	close(outputChan)

	if err != nil {
		log.Printf("Docker compose process exited with error: %v", err)
		log.Println("NOTE: This may be a temporary container failure. Docker Compose may still be managing restarts.")
	} else {
		log.Println("Docker compose process exited normally")
	}

	return nil
}

func (e *Executor) Stop() error {
	if e.cmd != nil && e.cmd.Process != nil {
		log.Println("Stopping docker compose process...")
		return e.cmd.Process.Signal(os.Interrupt)
	}
	return nil
}

func (e *Executor) StopAndCleanup() error {
	e.cleanupOnce.Do(func() {
		fmt.Println("\n=== StopAndCleanup called ===")

		// Kill the docker compose process forcefully to avoid conflict with cleanup
		if e.cmd != nil && e.cmd.Process != nil {
			fmt.Println("Killing docker compose process...")
			e.cmd.Process.Kill()
		}

		fmt.Println("Waiting 3 seconds for process to die...")
		time.Sleep(3 * time.Second)

		fmt.Println("Starting cleanup...")
		// Use CleanupAll (with output) so user can see what's happening
		e.cleanupErr = e.CleanupAll()
		fmt.Println("Cleanup finished")
	})
	return e.cleanupErr
}

func parseEnvVars(envString string) []string {
	if envString == "" {
		return []string{}
	}
	return strings.Fields(envString)
}
