package docker

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Executor struct {
	workDir     string
	edition     string
	cmd         *exec.Cmd
	cleanupOnce sync.Once
	cleanupErr  error
}

func NewExecutor(workDir string) *Executor {
	return &Executor{
		workDir: workDir,
		edition: "ce", // default to CE
	}
}

func (e *Executor) SetEdition(edition string) {
	e.edition = edition
}

func (e *Executor) getProjectName() string {
	if e.edition == "advanced" {
		return "carbonio-advanced"
	}
	return "carbonio"
}

func (e *Executor) CleanupExisting(edition string) error {
	return e.CleanupAll()
}

const lastEditionFile = ".last-edition"

// HasEditionChanged returns true if the current edition differs from the last
// one that was started. Returns false if no previous edition was recorded.
func (e *Executor) HasEditionChanged() bool {
	data, err := os.ReadFile(filepath.Join(e.workDir, lastEditionFile))
	if err != nil {
		return false // no previous record, nothing to force
	}
	prev := strings.TrimSpace(string(data))
	return prev != "" && prev != e.edition
}

// SaveLastEdition writes the current edition to disk so the next run can detect changes.
func (e *Executor) SaveLastEdition() {
	path := filepath.Join(e.workDir, lastEditionFile)
	if err := os.WriteFile(path, []byte(e.edition), 0644); err != nil {
		log.Printf("Warning: failed to save last edition: %v", err)
	}
}

func (e *Executor) CleanupAll() error {
	return e.cleanupAllWithOutput(true)
}

func (e *Executor) CleanupAllQuiet() error {
	return e.cleanupAllWithOutput(false)
}

var (
	dockerNeedsOverride     bool
	dockerNeedsOverrideOnce sync.Once
)

// NeedsPlatformOverride checks the Docker daemon architecture and returns true
// if it is NOT amd64/x86_64, meaning platform overrides are needed.
func NeedsPlatformOverride() bool {
	dockerNeedsOverrideOnce.Do(func() {
		cmd := exec.Command("docker", "info", "--format", "{{.Architecture}}")
		out, err := cmd.Output()
		if err != nil {
			log.Printf("Could not detect Docker architecture: %v, assuming override needed", err)
			dockerNeedsOverride = true
			return
		}
		arch := strings.TrimSpace(string(out))
		log.Printf("Docker daemon architecture: %s", arch)
		switch arch {
		case "x86_64", "amd64":
			dockerNeedsOverride = false
		default:
			dockerNeedsOverride = true
		}
	})
	return dockerNeedsOverride
}

func getDockerEnv() []string {
	env := os.Environ()
	if NeedsPlatformOverride() {
		env = append(env, "DOCKER_DEFAULT_PLATFORM=linux/amd64")
		log.Printf("Non-amd64 Docker detected, setting DOCKER_DEFAULT_PLATFORM=linux/amd64")
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

	// Clean both project names to ensure all containers are removed
	// regardless of which edition was used in the previous run
	projectNames := []string{"carbonio", "carbonio-advanced"}

	for _, projectName := range projectNames {
		fmt.Printf("Stopping containers for project %s...\n", projectName)
		stopCmd := e.createDockerCommand(
			"compose",
			"--project-name", projectName,
			"-f", "docker-compose.yaml",
			"-f", "docker-compose-advanced.yaml",
			"stop",
			"--timeout", "30",
		)
		if showOutput {
			stopCmd.Stdout = os.Stdout
			stopCmd.Stderr = os.Stderr
		}
		stopCmd.Run() // Ignore errors - project may not exist

		cmd := e.createDockerCommand(
			"compose",
			"--project-name", projectName,
			"-f", "docker-compose.yaml",
			"-f", "docker-compose-advanced.yaml",
			"down",
			"--remove-orphans",
			"--timeout", "30",
		)
		if showOutput {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		}
		cmd.Run() // Ignore errors - project may not exist
	}

	time.Sleep(2 * time.Second)

	log.Println("Removing consul data volume to prevent rejoin errors...")
	// Consul volumes use default naming: {project-name}_{volume-name}
	consulVolumeNames := []string{
		"carbonio_consul-data",
		"carbonio-advanced_consul-data-advanced",
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

// CleanAllVolumes removes all persistent volumes for a completely fresh start.
// Uses "docker compose down -v" which lets compose itself resolve volume names
// from the compose files, so nothing is hardcoded.
func (e *Executor) CleanAllVolumes() error {
	log.Println("Removing all persistent volumes for a fresh start...")

	projectNames := []string{"carbonio", "carbonio-advanced"}
	for _, projectName := range projectNames {
		log.Printf("Removing volumes for project %s...", projectName)
		cmd := e.createDockerCommand(
			"compose",
			"--project-name", projectName,
			"-f", "docker-compose.yaml",
			"-f", "docker-compose-advanced.yaml",
			"down", "-v",
			"--timeout", "5",
		)
		if err := cmd.Run(); err != nil {
			log.Printf("Volume cleanup for %s: %v (may not exist)", projectName, err)
		}
	}

	log.Println("Volume cleanup done")
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
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered from stdout goroutine panic: %v", r)
			}
		}()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			log.Println(line)
			outputChan <- line
		}
	}()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered from stderr goroutine panic: %v", r)
			}
		}()
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

func (e *Executor) StreamServiceLogs(ctx context.Context, serviceName string, tail int, outputChan chan string) error {
	projectName := e.getProjectName()
	cmd := e.createDockerCommand("compose", "--project-name", projectName, "logs", "-f", "--tail", strconv.Itoa(tail), serviceName)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		close(outputChan)
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	cmd.Stderr = cmd.Stdout // merge stderr into stdout

	if err := cmd.Start(); err != nil {
		close(outputChan)
		return fmt.Errorf("failed to start log stream: %w", err)
	}

	// Kill process when context is cancelled
	go func() {
		<-ctx.Done()
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()

	go func() {
		defer close(outputChan)
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			case outputChan <- scanner.Text():
			}
		}
		cmd.Wait()
	}()

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
