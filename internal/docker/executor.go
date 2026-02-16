package docker

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
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

// conflictingProject returns the compose project name of the OTHER edition.
func (e *Executor) conflictingProject() string {
	if e.edition == "advanced" {
		return "carbonio"
	}
	return "carbonio-advanced"
}

// getComposeVolumeNames returns the actual Docker volume names defined in the
// compose files for the given project, by parsing `docker compose config`.
// This correctly resolves both auto-prefixed names and hardcoded `name:` values.
func (e *Executor) getComposeVolumeNames(projectName string) []string {
	args := []string{
		"compose",
		"--project-name", projectName,
		"-f", "docker-compose.yaml",
	}
	// Advanced uses both compose files, CE uses only the base one
	if projectName == "carbonio-advanced" {
		args = append(args, "-f", "docker-compose-advanced.yaml")
	}
	args = append(args, "config", "--format", "json")

	cmd := e.createDockerCommand(args...)
	out, err := cmd.Output()
	if err != nil {
		log.Printf("Warning: could not get compose config for %s: %v", projectName, err)
		return nil
	}

	var config struct {
		Volumes map[string]struct {
			Name string `json:"name"`
		} `json:"volumes"`
	}
	if err := json.Unmarshal(out, &config); err != nil {
		log.Printf("Warning: could not parse compose config: %v", err)
		return nil
	}

	var names []string
	for _, vol := range config.Volumes {
		if vol.Name != "" {
			names = append(names, vol.Name)
		}
	}
	log.Printf("Resolved %d volume names for project %s: %v", len(names), projectName, names)
	return names
}

// HasConflictingVolumes checks whether Docker volumes from a different edition exist,
// by reading the actual volume names from the compose files.
func (e *Executor) HasConflictingVolumes() bool {
	project := e.conflictingProject()
	volumeNames := e.getComposeVolumeNames(project)
	for _, name := range volumeNames {
		cmd := exec.Command("docker", "volume", "inspect", name)
		if err := cmd.Run(); err == nil {
			log.Printf("Found conflicting volume: %s (from project %s)", name, project)
			return true
		}
	}
	return false
}

// CleanConflictingVolumes removes volumes and containers from the other edition.
func (e *Executor) CleanConflictingVolumes() error {
	project := e.conflictingProject()
	log.Printf("Cleaning conflicting edition: project %s...", project)

	// Build compose file args matching the conflicting edition
	composeArgs := []string{
		"compose",
		"--project-name", project,
		"-f", "docker-compose.yaml",
	}
	if project == "carbonio-advanced" {
		composeArgs = append(composeArgs, "-f", "docker-compose-advanced.yaml")
	}

	// Stop + down -v for the conflicting project
	stopCmd := e.createDockerCommand(append(composeArgs, "stop")...)
	stopCmd.Run()

	downCmd := e.createDockerCommand(append(composeArgs, "down", "-v", "--remove-orphans")...)
	if err := downCmd.Run(); err != nil {
		log.Printf("Compose down for %s: %v (may not exist)", project, err)
	}

	// Ensure all containers are actually gone
	e.forceRemoveProjectContainers([]string{project})

	// Remove any volumes that compose down -v may have missed (e.g. hardcoded names)
	for _, name := range e.getComposeVolumeNames(project) {
		rmCmd := exec.Command("docker", "volume", "rm", "-f", name)
		if err := rmCmd.Run(); err != nil {
			log.Printf("Warning: failed to remove volume %s: %v", name, err)
		} else {
			log.Printf("Removed conflicting volume: %s", name)
		}
	}

	log.Println("Conflicting edition cleanup done")
	return nil
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

// forceRemoveProjectContainers lists all containers belonging to the given
// project names and force-removes any that still exist. It loops until none
// remain so the caller can be certain everything is down.
func (e *Executor) forceRemoveProjectContainers(projectNames []string) {
	for _, project := range projectNames {
		for {
			cmd := exec.Command("docker", "ps", "-aq", "--filter", "label=com.docker.compose.project="+project)
			out, err := cmd.Output()
			if err != nil {
				log.Printf("Warning: could not list containers for project %s: %v", project, err)
				break
			}
			ids := strings.Fields(strings.TrimSpace(string(out)))
			if len(ids) == 0 {
				break
			}
			log.Printf("Force-removing %d remaining containers for project %s...", len(ids), project)
			args := append([]string{"rm", "-f"}, ids...)
			rmCmd := exec.Command("docker", args...)
			rmCmd.Run()
			time.Sleep(1 * time.Second)
		}
	}
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
		)
		if showOutput {
			stopCmd.Stdout = os.Stdout
			stopCmd.Stderr = os.Stderr
		}
		stopCmd.Run()

		cmd := e.createDockerCommand(
			"compose",
			"--project-name", projectName,
			"-f", "docker-compose.yaml",
			"-f", "docker-compose-advanced.yaml",
			"down",
			"--remove-orphans",
		)
		if showOutput {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		}
		cmd.Run()
	}

	// Ensure everything is actually gone
	e.forceRemoveProjectContainers(projectNames)

	log.Println("Removing consul data volume to prevent rejoin errors...")
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
	pruneCmd := e.createDockerCommand("system", "prune", "-f")
	if showOutput {
		pruneCmd.Stdout = os.Stdout
		pruneCmd.Stderr = os.Stderr
	}
	if err := pruneCmd.Run(); err != nil {
		log.Printf("System prune failed: %v", err)
	} else {
		log.Println("System prune completed")
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
			"--remove-orphans",
		)
		if err := cmd.Run(); err != nil {
			log.Printf("Volume cleanup for %s: %v (may not exist)", projectName, err)
		}
	}

	e.forceRemoveProjectContainers(projectNames)

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
		scanner.Split(scanLinesOrCR)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.TrimSpace(line) == "" {
				continue
			}
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
		scanner.Split(scanLinesOrCR)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.TrimSpace(line) == "" {
				continue
			}
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

// scanLinesOrCR is a bufio.SplitFunc that splits on \n, \r\n, or \r.
// Docker Compose uses \r for in-place progress updates (pull progress);
// the default bufio.ScanLines only splits on \n.
func scanLinesOrCR(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	// Find the earliest \r or \n
	idxN := bytes.IndexByte(data, '\n')
	idxR := bytes.IndexByte(data, '\r')

	idx := -1
	if idxN >= 0 && idxR >= 0 {
		if idxR < idxN {
			idx = idxR
		} else {
			idx = idxN
		}
	} else if idxN >= 0 {
		idx = idxN
	} else if idxR >= 0 {
		idx = idxR
	}

	if idx >= 0 {
		advance = idx + 1
		// Handle \r\n as a single delimiter
		if data[idx] == '\r' && idx+1 < len(data) && data[idx+1] == '\n' {
			advance = idx + 2
		}
		return advance, data[:idx], nil
	}

	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
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
