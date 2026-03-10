package docker

import (
	"bufio"
	"bytes"
	"context"
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

	"github.com/creack/pty"
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

// Reset clears one-shot state so the executor can be reused for a new session
// (e.g. after navigating back to the home screen without restarting the app).
func (e *Executor) Reset() {
	e.cmd = nil
	e.cleanupOnce = sync.Once{}
	e.cleanupErr = nil
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
	projectName := e.getProjectName()
	log.Printf("Running cleanup for project %s...", projectName)

	composeArgs := []string{
		"compose",
		"--project-name", projectName,
		"-f", "docker-compose.yaml",
	}
	if e.edition == "advanced" {
		composeArgs = append(composeArgs, "-f", "docker-compose-advanced.yaml")
	}

	fmt.Printf("Stopping containers for project %s...\n", projectName)
	stopCmd := e.createDockerCommand(append(composeArgs, "stop")...)
	if showOutput {
		stopCmd.Stdout = os.Stdout
		stopCmd.Stderr = os.Stderr
	}
	stopCmd.Run()

	downCmd := e.createDockerCommand(append(composeArgs, "down", "--remove-orphans")...)
	if showOutput {
		downCmd.Stdout = os.Stdout
		downCmd.Stderr = os.Stderr
	}
	downCmd.Run()

	e.forceRemoveProjectContainers([]string{projectName})

	// Remove consul data volume to prevent rejoin errors
	consulVolName := "carbonio_consul-data"
	if e.edition == "advanced" {
		consulVolName = "carbonio-advanced_consul-data-advanced"
	}
	rmCmd := e.createDockerCommand("volume", "rm", "-f", consulVolName)
	if err := rmCmd.Run(); err != nil {
		log.Printf("Volume %s removal: %v (may not exist, this is fine)", consulVolName, err)
	} else {
		log.Printf("Volume %s removed successfully", consulVolName)
	}

	if showOutput {
		log.Println("Running docker system prune...")
		pruneCmd := e.createDockerCommand("system", "prune", "-f")
		pruneCmd.Stdout = os.Stdout
		pruneCmd.Stderr = os.Stderr
		if err := pruneCmd.Run(); err != nil {
			log.Printf("System prune failed: %v", err)
		} else {
			log.Println("System prune completed")
		}
	} else {
		log.Println("Skipping system prune (initial cleanup)")
	}

	log.Println("Cleanup completed")
	return nil
}

// CleanAllVolumes removes persistent volumes for the current edition.
func (e *Executor) CleanAllVolumes() error {
	projectName := e.getProjectName()
	log.Printf("Removing persistent volumes for project %s...", projectName)

	args := []string{
		"compose",
		"--project-name", projectName,
		"-f", "docker-compose.yaml",
	}
	if e.edition == "advanced" {
		args = append(args, "-f", "docker-compose-advanced.yaml")
	}
	args = append(args, "down", "-v", "--remove-orphans")

	cmd := e.createDockerCommand(args...)
	if err := cmd.Run(); err != nil {
		log.Printf("Volume cleanup for %s: %v (may not exist)", projectName, err)
	}

	e.forceRemoveProjectContainers([]string{projectName})

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

// PullImage pulls a single Docker image using a PTY so that Docker emits
// progress output (Downloading current/total). All non-empty lines are sent
// to outputChan. The caller must close the channel after this returns.
func (e *Executor) PullImage(ctx context.Context, imageRef string, outputChan chan<- string) error {
	cmd := exec.CommandContext(ctx, "docker", "pull", imageRef)
	cmd.Dir = e.workDir
	cmd.Env = getDockerEnv()

	// Wide terminal so Docker shows full progress bars with current/total
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 24, Cols: 200})
	if err != nil {
		return fmt.Errorf("docker pull %s: %w", imageRef, err)
	}
	defer ptmx.Close()

	scanner := bufio.NewScanner(ptmx)
	scanner.Split(scanLinesOrCR)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) != "" {
			select {
			case outputChan <- line:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	// PTY read error on process exit is expected — ignore scanner.Err()
	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("docker pull %s: %w", imageRef, err)
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
		log.Println("StopAndCleanup called")

		// Kill the docker compose process forcefully to avoid conflict with cleanup
		if e.cmd != nil && e.cmd.Process != nil {
			log.Println("Killing docker compose process...")
			e.cmd.Process.Kill()
		}

		log.Println("Waiting 3 seconds for process to die...")
		time.Sleep(3 * time.Second)

		log.Println("Starting cleanup...")
		e.cleanupErr = e.CleanupAll()
		log.Println("Cleanup finished")
	})
	return e.cleanupErr
}

func parseEnvVars(envString string) []string {
	if envString == "" {
		return []string{}
	}
	return strings.Fields(envString)
}
