package gui

import (
	"carbonio-docker-cli/internal/provisioner"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type serviceState int

const (
	stateUnknown serviceState = iota
	statePulling
	stateCreating
	stateStarting
	stateRunning
	stateError
)

func (s serviceState) String() string {
	switch s {
	case statePulling:
		return "Pulling"
	case stateCreating:
		return "Creating"
	case stateStarting:
		return "Starting"
	case stateRunning:
		return "Running"
	case stateError:
		return "Error"
	default:
		return "Waiting"
	}
}

type dockerComposeStatus struct {
	Name    string `json:"Name"`
	State   string `json:"State"`
	Status  string `json:"Status"`
	Health  string `json:"Health"`
	Service string `json:"Service"`
}

func (a *App) ShowMonitorScreen(envVars string, cmdParts []string, visibleServices []string) {
	ctx, cancel := context.WithCancel(context.Background())

	states := make(map[string]serviceState)
	for _, svc := range visibleServices {
		states[svc] = stateUnknown
	}
	var mu sync.Mutex

	// Parse provisioned accounts
	accounts, err := provisioner.ParseProvisioningScript(a.workDir)
	if err != nil {
		log.Printf("Warning: failed to parse provisioning script: %v", err)
		accounts = []provisioner.Account{}
	}

	// Accounts panel
	accountsBox := container.NewVBox()
	if len(accounts) > 0 {
		accountsBox.Add(widget.NewRichTextFromMarkdown("### Available Accounts"))
		for _, acc := range accounts {
			adminBadge := ""
			if acc.IsAdmin {
				adminBadge = " [ADMIN]"
			}
			accountsBox.Add(widget.NewLabel(fmt.Sprintf("  %s / %s%s", acc.Username, acc.Password, adminBadge)))
		}
		accountsBox.Add(widget.NewSeparator())
	}

	// Service states list
	servicesList := widget.NewList(
		func() int {
			return len(visibleServices)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				container.NewHBox(), // placeholder for badge
				widget.NewLabel("service-name-placeholder"),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			row := obj.(*fyne.Container)
			mu.Lock()
			sortedNames := make([]string, len(visibleServices))
			copy(sortedNames, visibleServices)
			sort.Strings(sortedNames)
			name := sortedNames[id]
			state := states[name]
			mu.Unlock()

			badge := NewStatusBadge(state.String())
			nameLabel := widget.NewLabel(name)

			row.Objects = []fyne.CanvasObject{badge, nameLabel}
			row.Refresh()
		},
	)
	servicesList.OnSelected = func(_ widget.ListItemID) {
		servicesList.UnselectAll()
	}

	// Log output
	logEntry := widget.NewMultiLineEntry()
	logEntry.Disable()
	logEntry.SetMinRowsVisible(10)

	showLogs := false
	logContainer := container.NewStack(logEntry)
	logContainer.Hide()

	toggleLogsBtn := widget.NewButton("Show Logs", func() {
		showLogs = !showLogs
		if showLogs {
			logContainer.Show()
		} else {
			logContainer.Hide()
		}
	})

	cleaning := false
	stopBtn := widget.NewButton("Stop & Cleanup", func() {
		if cleaning {
			return
		}
		cleaning = true
		cancel()
		prog := dialog.NewProgressInfinite("Stopping", "Stopping and cleaning up containers...", a.window)
		prog.Show()
		go func() {
			if err := a.executor.StopAndCleanup(); err != nil {
				log.Printf("Cleanup error: %v", err)
			}
			prog.Hide()
			showSuccessDialog("Cleanup Complete", "All containers have been stopped and cleaned up.", a.window)
		}()
	})
	stopBtn.Importance = widget.DangerImportance

	title := widget.NewRichTextFromMarkdown("# Carbonio Services Monitor")

	topSection := container.NewVBox(title, accountsBox)
	bottomSection := container.NewVBox(
		widget.NewSeparator(),
		container.NewHBox(toggleLogsBtn, layout.NewSpacer(), stopBtn),
	)

	content := container.NewBorder(
		topSection,
		container.NewVBox(logContainer, bottomSection),
		nil, nil,
		servicesList,
	)

	a.window.SetContent(content)

	// Start docker compose
	outputChan := make(chan string, 100)

	go func() {
		err := a.executor.Execute(envVars, cmdParts, outputChan)
		if err != nil {
			log.Printf("Docker execution error: %v", err)
		}
	}()

	// Goroutine 1: read output, append to log
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case line, ok := <-outputChan:
				if !ok {
					return
				}
				logEntry.SetText(logEntry.Text + line + "\n")
				logEntry.CursorRow = len(strings.Split(logEntry.Text, "\n")) - 1

				// Parse log line for state hints
				parseLogForStates(line, visibleServices, states, &mu)
				servicesList.Refresh()
			}
		}
	}()

	// Goroutine 2: poll docker compose ps every 2s
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				newStates := fetchDockerStates(a.workDir, a.getProjectName(), visibleServices)
				mu.Lock()
				for k, v := range newStates {
					states[k] = v
				}
				mu.Unlock()
				servicesList.Refresh()
			}
		}
	}()

	// Cleanup on window close
	a.window.SetCloseIntercept(func() {
		if !cleaning {
			cleaning = true
			cancel()
			a.executor.StopAndCleanup()
		}
		a.window.Close()
	})
}

func fetchDockerStates(workDir, projectName string, trackedServices []string) map[string]serviceState {
	result := make(map[string]serviceState)

	cmd := exec.Command("docker", "compose", "--project-name", projectName, "ps", "--format", "json")
	cmd.Dir = workDir
	output, err := cmd.Output()
	if err != nil {
		return result
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var status dockerComposeStatus
		if err := json.Unmarshal([]byte(line), &status); err != nil {
			continue
		}
		serviceName := status.Service
		if serviceName == "" {
			continue
		}

		tracked := false
		for _, svc := range trackedServices {
			if svc == serviceName {
				tracked = true
				break
			}
		}
		if !tracked {
			continue
		}

		state := strings.ToLower(status.State)
		var newState serviceState

		switch state {
		case "running":
			if status.Health == "unhealthy" {
				newState = stateError
			} else {
				newState = stateRunning
			}
		case "created", "restarting":
			newState = stateStarting
		case "paused", "dead":
			newState = stateError
		case "exited":
			if strings.Contains(status.Status, "Exited (0)") {
				newState = stateRunning
			} else {
				newState = stateError
			}
		default:
			newState = stateUnknown
		}

		result[serviceName] = newState
	}

	return result
}

func parseLogForStates(line string, services []string, states map[string]serviceState, mu *sync.Mutex) {
	lineLower := strings.ToLower(line)
	if strings.Contains(line, "|") {
		return
	}

	for _, serviceName := range services {
		if !strings.Contains(lineLower, strings.ToLower(serviceName)) {
			continue
		}

		var newState serviceState
		if strings.Contains(lineLower, "pulling") || strings.Contains(lineLower, "pull") {
			newState = statePulling
		} else if strings.Contains(lineLower, "creating") {
			newState = stateCreating
		} else if strings.Contains(lineLower, "created") || strings.Contains(lineLower, "starting") {
			newState = stateStarting
		} else if strings.Contains(lineLower, "started") {
			newState = stateRunning
		} else if strings.Contains(lineLower, "error") {
			newState = stateError
		} else {
			continue
		}

		mu.Lock()
		states[serviceName] = newState
		mu.Unlock()
	}
}
