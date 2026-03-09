package gui

import (
	"carbonio-dockerization-tools/internal/docker"
	"carbonio-dockerization-tools/internal/parser"
	"carbonio-dockerization-tools/internal/provisioner"
	"context"
	"encoding/json"
	"fmt"
	"image/color"
	"log"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type serviceState int

const (
	stateQueued serviceState = iota
	statePulling
	stateReady
	stateStarting
	stateRunning
	stateFailed
	stateStopping
)

const colorNameReady fyne.ThemeColorName = "stateReady"

const maxLogLines = 500

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\].*?\x07|\x1b[^[\]]*`)

func stripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

func (s serviceState) String() string {
	switch s {
	case stateQueued:
		return "Queued"
	case statePulling:
		return "Pulling"
	case stateReady:
		return "Ready"
	case stateStarting:
		return "Starting"
	case stateRunning:
		return "Running"
	case stateFailed:
		return "Failed"
	case stateStopping:
		return "Stopping"
	default:
		return "Queued"
	}
}

func (s serviceState) ColorName() fyne.ThemeColorName {
	switch s {
	case statePulling:
		return theme.ColorNameWarning
	case stateReady, stateStarting:
		return colorNameReady
	case stateRunning:
		return theme.ColorNameSuccess
	case stateFailed:
		return theme.ColorNameError
	default:
		return theme.ColorNamePlaceHolder
	}
}

type dockerComposeStatus struct {
	Name    string `json:"Name"`
	State   string `json:"State"`
	Status  string `json:"Status"`
	Health  string `json:"Health"`
	Service string `json:"Service"`
}

type monitoredService struct {
	name        string
	expanded    bool
	cancel      context.CancelFunc
	logEntry    *widget.Entry
	logLines    int
	logBox      *fyne.Container
	toggleBtn   *widget.Button
	statusLabel *widget.RichText
}

func (ms *monitoredService) updateStatus(state serviceState) {
	ms.statusLabel.Segments = []widget.RichTextSegment{
		&widget.TextSegment{
			Text: "(" + state.String() + ")",
			Style: widget.RichTextStyle{
				SizeName:  theme.SizeNameCaptionText,
				ColorName: state.ColorName(),
			},
		},
	}
	ms.statusLabel.Refresh()
}

func (a *App) ShowMonitorScreen(result *docker.BuildResult, visibleServices []string) {
	ctx, cancel := context.WithCancel(context.Background())

	states := make(map[string]serviceState)
	for _, svc := range visibleServices {
		states[svc] = stateQueued
	}
	isPullPhase := true
	var mu sync.Mutex

	sortedServices := make([]string, len(visibleServices))
	copy(sortedServices, visibleServices)
	sort.Strings(sortedServices)

	// Parse provisioned accounts
	accounts, err := provisioner.ParseProvisioningScript(a.workDir)
	if err != nil {
		log.Printf("Warning: failed to parse provisioning script: %v", err)
		accounts = []provisioner.Account{}
	}

	// Accounts panel
	accountsBox := container.NewVBox()
	if len(accounts) > 0 {
		accountsHeader := newSectionHeader("Available Accounts")
		accountsBox.Add(accountsHeader)
		for _, acc := range accounts {
			adminBadge := ""
			if acc.IsAdmin {
				adminBadge = " [ADMIN]"
			}
			accountsBox.Add(newCompactLabel(fmt.Sprintf("  %s / %s%s", acc.Username, acc.Password, adminBadge), false))
		}
		accountsBox.Add(widget.NewSeparator())
	}

	// Global status indicator (single RichText widget for perfect alignment)
	readyURL, _ := url.Parse("https://docker.carbonio.localhost")
	globalStatusLabel := widget.NewRichText(&widget.TextSegment{
		Text:  "Waiting...",
		Style: widget.RichTextStyle{SizeName: theme.SizeNameCaptionText, ColorName: theme.ColorNamePlaceHolder, TextStyle: fyne.TextStyle{Bold: true}},
	})
	setGlobalStatus := func(text string, colorName fyne.ThemeColorName, showLink bool) {
		if showLink {
			globalStatusLabel.Segments = []widget.RichTextSegment{
				&widget.TextSegment{
					Text:  text,
					Style: widget.RichTextStyle{SizeName: theme.SizeNameCaptionText, ColorName: colorName, TextStyle: fyne.TextStyle{Bold: true}},
				},
				&widget.HyperlinkSegment{
					Text: "https://docker.carbonio.localhost",
					URL:  readyURL,
				},
			}
		} else {
			globalStatusLabel.Segments = []widget.RichTextSegment{
				&widget.TextSegment{
					Text:  text,
					Style: widget.RichTextStyle{SizeName: theme.SizeNameCaptionText, ColorName: colorName, TextStyle: fyne.TextStyle{Bold: true}},
				},
			}
		}
		globalStatusLabel.Refresh()
	}

	// Build per-service expandable entries
	monitored := make(map[string]*monitoredService)
	servicesGrid := container.NewVBox()

	for _, name := range sortedServices {
		svc := &monitoredService{name: name}

		// Log entry (hidden by default) — not disabled so text stays white
		logEntry := widget.NewMultiLineEntry()
		logEntry.SetMinRowsVisible(8)
		svc.logEntry = logEntry

		logBox := container.NewStack(logEntry)
		logBox.Hide()
		svc.logBox = logBox

		statusLabel := widget.NewRichText(&widget.TextSegment{
			Text:  "(" + stateQueued.String() + ")",
			Style: widget.RichTextStyle{SizeName: theme.SizeNameCaptionText, ColorName: stateQueued.ColorName()},
		})
		svc.statusLabel = statusLabel

		monitored[name] = svc

		nameLabel := newCompactLabel(name, false)

		svcName := name // capture for closure
		toggleBtn := widget.NewButtonWithIcon("", theme.MenuDropDownIcon(), func() {
			ms := monitored[svcName]
			if ms.expanded {
				// Collapse
				ms.expanded = false
				if ms.cancel != nil {
					ms.cancel()
					ms.cancel = nil
				}
				ms.toggleBtn.SetIcon(theme.MenuDropDownIcon())
				ms.logBox.Hide()
			} else {
				// Expand
				ms.expanded = true
				ms.logEntry.SetText("")
				ms.logLines = 0
				ms.toggleBtn.SetIcon(theme.MenuDropUpIcon())
				ms.logBox.Show()

				logCtx, logCancel := context.WithCancel(ctx)
				ms.cancel = logCancel

				outputChan := make(chan string, 100)
				go func() {
					if err := a.executor.StreamServiceLogs(logCtx, svcName, 50, outputChan); err != nil {
						log.Printf("Failed to start log stream for %s: %v", svcName, err)
						return
					}
				}()

				go func() {
					for {
						select {
						case <-logCtx.Done():
							return
						case line, ok := <-outputChan:
							if !ok {
								return
							}
							cleaned := stripANSI(line)
							fyne.Do(func() {
								appendToServiceLog(ms, cleaned)
							})
						}
					}
				}()
			}
		})
		toggleBtn.Importance = widget.LowImportance
		toggleBtn.Hide()
		svc.toggleBtn = toggleBtn

		// Layout: [status(fixed)] name ............. [▼]
		statusSpacer := canvas.NewRectangle(color.Transparent)
		statusSpacer.SetMinSize(fyne.NewSize(70, 0))
		statusBox := container.NewStack(statusSpacer, statusLabel)
		headerRow := container.NewBorder(nil, nil,
			container.NewHBox(statusBox, nameLabel),
			toggleBtn,
			nil,
		)
		servicesGrid.Add(headerRow)
		servicesGrid.Add(logBox)
	}

	exportBtn := widget.NewButton("Export Config", func() {
		fd := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
			if err != nil {
				showErrorDialog(err.Error(), a.window)
				return
			}
			if writer == nil {
				return
			}
			writerPath := writer.URI().Path()
			writer.Close()
			os.Remove(writerPath)
			path := writerPath + ".carbonio-dockerization"
			if saveErr := a.saveConfig(path); saveErr != nil {
				showErrorDialog(saveErr.Error(), a.window)
				return
			}
			showSuccessDialog("Config Exported", fmt.Sprintf("Saved to:\n%s", path), a.window)
		}, a.window)
		fd.SetFileName("carbonio-config")
		fd.Show()
	})

	cleaning := false
	cleanDone := false

	doCleanup := func() {
		if cleaning {
			return
		}
		cleaning = true
		cancel()

		mu.Lock()
		for _, svc := range visibleServices {
			states[svc] = stateStopping
		}
		mu.Unlock()
		for _, ms := range monitored {
			ms.updateStatus(stateStopping)
		}
		setGlobalStatus("Stopping...", theme.ColorNameWarning, false)

		prog := showProgressModal("Stopping", "Stopping and cleaning up containers...", a.window)
		go func() {
			if err := a.executor.StopAndCleanup(); err != nil {
				log.Printf("Cleanup error: %v", err)
			}
			fyne.Do(func() {
				cleanDone = true
				prog.Hide()
				setGlobalStatus("Stopped", theme.ColorNameForeground, false)
				showCleanupCompleteDialog(a.window, func() {
					a.window.Close()
				})
			})
		}()
	}

	copyBtn := widget.NewButton("Copy startup command", func() {
		a.window.Clipboard().SetContent(result.StartupCommand())
		showSuccessDialog("Copied", "Startup command copied to clipboard.", a.window)
	})

	stopBtn := widget.NewButton("Stop & Cleanup", func() {
		doCleanup()
	})
	stopBtn.Importance = widget.HighImportance

	// Header
	logo := newLogo(32)
	title := widget.NewRichTextFromMarkdown("# Carbonio Services Monitor")
	titleRow := container.NewHBox(logo, title)

	editionLabel := "CE (Community Edition)"
	if a.edition == parser.EditionAdvanced {
		editionLabel = "Advanced Edition"
	}
	editionSubtitle := widget.NewRichText(&widget.TextSegment{
		Text: editionLabel,
		Style: widget.RichTextStyle{
			SizeName:  theme.SizeNameSubHeadingText,
			TextStyle: fyne.TextStyle{Bold: true},
		},
	})

	servicesHeaderLabel := newSectionHeader("Services")
	servicesHeaderRow := container.NewHBox(servicesHeaderLabel, layout.NewSpacer(), globalStatusLabel)

	topSection := container.NewPadded(container.NewPadded(container.NewVBox(
		titleRow, editionSubtitle, widget.NewSeparator(),
		accountsBox,
		servicesHeaderRow,
	)))

	bottomSection := container.NewPadded(container.NewPadded(container.NewHBox(
		wideButton(exportBtn, 150),
		layout.NewSpacer(),
		wideButton(copyBtn, 160),
		wideButton(stopBtn, 150),
	)))

	content := container.NewBorder(
		topSection,
		bottomSection,
		nil, nil,
		container.NewVScroll(container.NewPadded(servicesGrid)),
	)

	a.window.SetContent(content)

	updateGlobalStatus := func() {
		// Must be called with mu held
		hasError := false
		hasPulling := false
		hasReady := false
		allRunning := true

		for _, svc := range visibleServices {
			st := states[svc]
			switch st {
			case stateRunning:
				// ok
			case statePulling:
				hasPulling = true
				allRunning = false
			case stateReady, stateStarting:
				hasReady = true
				allRunning = false
			case stateFailed:
				hasError = true
				allRunning = false
			default:
				allRunning = false
			}
		}

		if hasError {
			setGlobalStatus("Failing", theme.ColorNameError, false)
		} else if allRunning {
			setGlobalStatus("Ready: ", theme.ColorNameSuccess, true)
		} else if hasPulling {
			setGlobalStatus("Pulling...", theme.ColorNameWarning, false)
		} else if hasReady {
			setGlobalStatus("Starting...", theme.ColorNameWarning, false)
		} else {
			setGlobalStatus("Waiting...", theme.ColorNamePlaceHolder, false)
		}
	}

	refreshBadges := func() {
		mu.Lock()
		defer mu.Unlock()
		for name, ms := range monitored {
			st := states[name]
			ms.updateStatus(st)
			if st == stateRunning || st == stateFailed {
				ms.toggleBtn.Show()
			}
		}
		updateGlobalStatus()
	}

	// Start docker compose: pull each image with progress, then up
	outputChan := make(chan string, 100)

	go func() {
		// Pull step: parse compose output to track per-service pull state
		pullChan := make(chan string, 100)
		go func() {
			for line := range pullChan {
				cleaned := stripANSI(strings.TrimSpace(line))
				if cleaned == "" {
					continue
				}
				lower := strings.ToLower(cleaned)
				for _, svcName := range visibleServices {
					if !strings.Contains(lower, strings.ToLower(svcName)) {
						continue
					}
					if strings.Contains(lower, "pulled") {
						mu.Lock()
						states[svcName] = stateReady
						mu.Unlock()
						fyne.Do(func() {
							refreshBadges()
						})
					} else if strings.Contains(lower, "pulling") {
						mu.Lock()
						states[svcName] = statePulling
						mu.Unlock()
						fyne.Do(func() {
							refreshBadges()
						})
					}
					break
				}
				outputChan <- cleaned
			}
		}()
		log.Println("Running docker compose pull...")
		if err := a.executor.Execute(result.EnvVars, result.PullCmd, pullChan); err != nil {
			log.Printf("Pull step error (non-fatal): %v", err)
		}

		// End pull phase: mark remaining queued/pulling services as ready (cached/local)
		mu.Lock()
		isPullPhase = false
		for _, svc := range visibleServices {
			if states[svc] == stateQueued || states[svc] == statePulling {
				states[svc] = stateReady
			}
		}
		mu.Unlock()
		fyne.Do(func() {
			refreshBadges()
		})

		// Up step: transition all ready services to starting
		mu.Lock()
		for _, svc := range visibleServices {
			if states[svc] == stateReady {
				states[svc] = stateStarting
			}
		}
		mu.Unlock()
		fyne.Do(func() {
			refreshBadges()
		})

		log.Println("Running docker compose up...")
		err := a.executor.Execute(result.EnvVars, result.UpCmd, outputChan)
		if err != nil {
			log.Printf("Docker execution error: %v", err)
		}
	}()

	// Goroutine: read compose output for state parsing
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case line, ok := <-outputChan:
				if !ok {
					return
				}
				parseLogForStates(line, visibleServices, states, &mu)
				fyne.Do(func() {
					refreshBadges()
				})
			}
		}
	}()

	// Goroutine: poll docker compose ps every 2s (skip during pull phase)
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				mu.Lock()
				pulling := isPullPhase
				mu.Unlock()
				if pulling {
					continue
				}
				newStates := fetchDockerStates(a.workDir, a.getProjectName(), visibleServices)
				mu.Lock()
				for k, v := range newStates {
					states[k] = v
				}
				mu.Unlock()
				fyne.Do(func() {
					refreshBadges()
				})
			}
		}
	}()

	// Cleanup on window close
	a.window.SetCloseIntercept(func() {
		if cleanDone {
			// Cleanup already finished, allow close
			a.window.SetCloseIntercept(nil)
			a.window.Close()
			return
		}
		if cleaning {
			// Cleanup in progress, ignore close request
			return
		}
		// Running: start cleanup (which will show the modal and close on OK)
		doCleanup()
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
				newState = stateFailed
			} else {
				newState = stateRunning
			}
		case "created", "restarting":
			newState = stateStarting
		case "paused", "dead":
			newState = stateFailed
		case "exited":
			if strings.Contains(status.Status, "Exited (0)") {
				newState = stateRunning
			} else {
				newState = stateFailed
			}
		default:
			newState = stateQueued
		}

		result[serviceName] = newState
	}

	return result
}

// appendToServiceLog appends a line to a service's log entry, trimming old lines if needed.
// Must be called from the UI thread (inside fyne.Do).
func appendToServiceLog(ms *monitoredService, line string) {
	ms.logLines++
	if ms.logLines > maxLogLines {
		text := ms.logEntry.Text
		idx := strings.Index(text, "\n")
		if idx >= 0 {
			text = text[idx+1:]
		}
		ms.logEntry.SetText(text + line + "\n")
	} else {
		ms.logEntry.SetText(ms.logEntry.Text + line + "\n")
	}
	ms.logEntry.CursorRow = ms.logLines
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
		if strings.Contains(lineLower, "pulled") {
			newState = stateReady
		} else if strings.Contains(lineLower, "pulling") || strings.Contains(lineLower, "pull") {
			newState = statePulling
		} else if strings.Contains(lineLower, "started") {
			newState = stateRunning
		} else if strings.Contains(lineLower, "creating") || strings.Contains(lineLower, "created") || strings.Contains(lineLower, "starting") {
			newState = stateStarting
		} else if strings.Contains(lineLower, "error") {
			newState = stateFailed
		} else {
			continue
		}

		mu.Lock()
		states[serviceName] = newState
		mu.Unlock()
	}
}
