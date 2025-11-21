package docker

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
)

// Executor handles docker compose execution
type Executor struct {
	workDir string
	cmd     *exec.Cmd
}

// NewExecutor creates a new executor
func NewExecutor(workDir string) *Executor {
	return &Executor{
		workDir: workDir,
	}
}

// CleanupExisting runs docker compose down to clean up any existing containers
// DEPRECATED: Use CleanupAll instead
func (e *Executor) CleanupExisting(edition string) error {
	return e.CleanupAll()
}

// CleanupAll runs docker compose down for BOTH CE and Advanced
// to ensure complete cleanup regardless of what's running
func (e *Executor) CleanupAll() error {
	log.Println("Running complete cleanup (CE + Advanced)...")

	args := []string{
		"compose",
		"-f", "docker-compose.yaml",
		"-f", "docker-compose-advanced.yaml",
		"down",
		"--remove-orphans", // Remove orphaned containers
	}

	cmd := exec.Command("docker", args...)
	cmd.Dir = e.workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Printf("Cleanup command failed: %v", err)
		return fmt.Errorf("cleanup failed: %w", err)
	}

	log.Println("Cleanup completed successfully")
	return nil
}

// Execute runs the docker compose command with environment variables
func (e *Executor) Execute(envVars string, cmdParts []string, outputChan chan string) error {
	// Create command
	cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
	cmd.Dir = e.workDir

	// Set environment variables
	cmd.Env = append(os.Environ(), parseEnvVars(envVars)...)

	// Setup pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start docker compose: %w", err)
	}

	e.cmd = cmd

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Signal received in executor, stopping and cleaning up...")
		e.StopAndCleanup()
	}()

	// Stream stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			log.Println(line)
			outputChan <- line
		}
	}()

	// Stream stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			log.Println(line)
			outputChan <- line
		}
	}()

	// Wait for command to finish
	err = cmd.Wait()
	close(outputChan)

	return err
}

// Stop stops the running docker compose command
func (e *Executor) Stop() error {
	if e.cmd != nil && e.cmd.Process != nil {
		log.Println("Stopping docker compose process...")
		return e.cmd.Process.Signal(os.Interrupt)
	}
	return nil
}

// StopAndCleanup stops the running command and does cleanup
func (e *Executor) StopAndCleanup() error {
	log.Println("Stopping and cleaning up...")

	// Stop running process
	if err := e.Stop(); err != nil {
		log.Printf("Failed to stop process: %v", err)
	}

	// Do full cleanup
	return e.CleanupAll()
}

// parseEnvVars parses space-separated env vars "KEY=value KEY2=value2"
func parseEnvVars(envString string) []string {
	if envString == "" {
		return []string{}
	}

	return strings.Fields(envString)
}
