package docker

import (
	"bufio"
	"fmt"
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
func (e *Executor) CleanupExisting(edition string) error {
	args := []string{"compose", "-f", "docker-compose.yaml"}

	if edition == "advanced" {
		args = append(args, "-f", "docker-compose-advanced.yaml")
	}

	args = append(args, "down")

	cmd := exec.Command("docker", args...)
	cmd.Dir = e.workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
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
		e.Stop()
	}()

	// Stream output
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			outputChan <- scanner.Text()
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			outputChan <- "[ERROR] " + scanner.Text()
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
		return e.cmd.Process.Signal(os.Interrupt)
	}
	return nil
}

// parseEnvVars parses space-separated env vars "KEY=value KEY2=value2"
func parseEnvVars(envString string) []string {
	if envString == "" {
		return []string{}
	}

	vars := strings.Fields(envString)
	return vars
}
