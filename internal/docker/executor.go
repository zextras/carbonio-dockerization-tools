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

type Executor struct {
	workDir string
	cmd     *exec.Cmd
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
func (e *Executor) cleanupAllWithOutput(showOutput bool) error {
	log.Println("Running complete cleanup (CE + Advanced)...")
	args := []string{
		"compose",
		"-f", "docker-compose.yaml",
		"-f", "docker-compose-advanced.yaml",
		"down",
		"--remove-orphans",
	}
	cmd := exec.Command("docker", args...)
	cmd.Dir = e.workDir
	if showOutput {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	if err := cmd.Run(); err != nil {
		log.Printf("Compose down failed: %v", err)
	} else {
		log.Println("Compose down completed")
	}
	log.Println("Running docker system prune...")
	pruneArgs := []string{
		"system",
		"prune",
		"-f",
	}
	pruneCmd := exec.Command("docker", pruneArgs...)
	pruneCmd.Dir = e.workDir
	if showOutput {
		pruneCmd.Stdout = os.Stdout
		pruneCmd.Stderr = os.Stderr
	}
	if err := pruneCmd.Run(); err != nil {
		log.Printf("System prune failed: %v", err)
		return fmt.Errorf("system prune failed: %w", err)
	}
	log.Println("System prune completed")
	log.Println("Cleanup completed successfully")
	return nil
}
func (e *Executor) Execute(envVars string, cmdParts []string, outputChan chan string) error {
	cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
	cmd.Dir = e.workDir
	cmd.Env = append(os.Environ(), parseEnvVars(envVars)...)
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
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Signal received in executor, stopping and cleaning up...")
		e.StopAndCleanup()
	}()
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
	return err
}
func (e *Executor) Stop() error {
	if e.cmd != nil && e.cmd.Process != nil {
		log.Println("Stopping docker compose process...")
		return e.cmd.Process.Signal(os.Interrupt)
	}
	return nil
}
func (e *Executor) StopAndCleanup() error {
	log.Println("Stopping and cleaning up...")
	if err := e.Stop(); err != nil {
		log.Printf("Failed to stop process: %v", err)
	}
	return e.CleanupAllQuiet()
}
func parseEnvVars(envString string) []string {
	if envString == "" {
		return []string{}
	}
	return strings.Fields(envString)
}
