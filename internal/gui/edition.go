package gui

import (
	"carbonio-docker-cli/internal/graph"
	"carbonio-docker-cli/internal/parser"
	"image/color"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func (a *App) ShowEditionScreen() {
	choice := widget.NewRadioGroup([]string{
		"CE (Community Edition)",
		"Advanced",
	}, nil)
	choice.SetSelected("CE (Community Edition)")

	nextBtn := widget.NewButton("Next", func() {
		edition := parser.EditionCE
		if choice.Selected == "Advanced" {
			edition = parser.EditionAdvanced
		}
		a.edition = edition
		a.executor.SetEdition(string(edition))

		parsedConfig, err := parser.ParseAll(a.workDir, edition)
		if err != nil {
			showErrorDialog(err.Error(), a.window)
			return
		}
		a.parsedConfig = parsedConfig
		resolver := graph.NewDependencyResolver(parsedConfig.BackendServices)
		a.ShowServicesScreen(resolver)
	})
	nextBtn.Importance = widget.HighImportance

	backBtn := widget.NewButton("Back", func() {
		a.ShowStartupScreen()
	})

	logo := newLogo(96)
	title := widget.NewRichTextFromMarkdown("# Carbonio Docker GUI")

	modeLabel := widget.NewRichText(&widget.TextSegment{
		Text: "Select edition:",
		Style: widget.RichTextStyle{
			SizeName:  theme.SizeNameCaptionText,
			ColorName: theme.ColorNamePlaceHolder,
		},
	})

	content := container.NewVBox(
		container.NewCenter(logo),
		title,
		container.NewPadded(container.NewVBox(
			modeLabel,
			choice,
		)),
		widget.NewSeparator(),
		container.NewGridWithColumns(2, wideButton(backBtn, 120), wideButton(nextBtn, 120)),
	)

	a.window.SetContent(container.NewCenter(content))
	a.window.Canvas().Focus(nil)
}

func (a *App) getProjectName() string {
	if a.edition == parser.EditionAdvanced {
		return "carbonio-advanced"
	}
	return "carbonio"
}

func (a *App) buildVisibleServicesList() []string {
	var visible []string
	for serviceName := range a.pendingBackend {
		if !parser.GlobalDockerConfig.IsServiceHidden(serviceName) {
			visible = append(visible, serviceName)
		}
	}
	return visible
}

func (a *App) promptCleanPersistenceThenMonitor(envVars string, cmdParts []string) {
	// If edition changed since last run, force clean without asking
	if a.executor.HasEditionChanged() {
		log.Println("Edition changed, forcing persistence cleanup")
		a.cleanPersistence = true
		a.startMonitor(envVars, cmdParts)
		return
	}

	titleLabel := widget.NewRichText(&widget.TextSegment{
		Text: "Clean All Persistence",
		Style: widget.RichTextStyle{
			SizeName:  theme.SizeNameSubHeadingText,
			TextStyle: fyne.TextStyle{Bold: true},
		},
	})
	messageLabel := widget.NewLabel(
		"Do you want to start with a fresh installation?\n\n" +
			"All persistent data from previous runs will be removed.")

	yesBtn := widget.NewButton("Yes, clean all", nil)
	noBtn := widget.NewButton("No, keep data", nil)
	noBtn.Importance = widget.HighImportance

	minWidth := canvas.NewRectangle(color.Transparent)
	minWidth.SetMinSize(fyne.NewSize(500, 0))

	inner := container.NewVBox(
		minWidth,
		titleLabel,
		widget.NewSeparator(),
		messageLabel,
		widget.NewSeparator(),
		container.NewGridWithColumns(2, yesBtn, noBtn),
	)

	bg := canvas.NewRectangle(theme.OverlayBackgroundColor())
	bg.CornerRadius = 8
	card := container.NewStack(bg, container.NewPadded(inner))
	pop := widget.NewModalPopUp(card, a.window.Canvas())

	yesBtn.OnTapped = func() {
		pop.Hide()
		a.cleanPersistence = true
		a.startMonitor(envVars, cmdParts)
	}
	noBtn.OnTapped = func() {
		pop.Hide()
		a.cleanPersistence = false
		a.startMonitor(envVars, cmdParts)
	}

	pop.Show()
}

func (a *App) handleConfigImport(filePath string) {
	err := a.loadConfigFromFile(filePath)
	if err != nil {
		showErrorDialog(err.Error(), a.window)
		return
	}

	envVars, cmdParts, err := a.buildDockerCommand()
	if err != nil {
		showErrorDialog(err.Error(), a.window)
		return
	}

	a.promptCleanPersistenceThenMonitor(envVars, cmdParts)
}

func (a *App) startMonitor(envVars string, cmdParts []string) {
	visibleServices := a.buildVisibleServicesList()
	a.executor.SaveLastEdition()
	if a.cleanPersistence {
		prog := showProgressModal("Cleaning Persistence",
			"Removing all persistent volumes for a fresh start...", a.window)
		go func() {
			if err := a.executor.CleanAllVolumes(); err != nil {
				log.Printf("Clean volumes error: %v", err)
			}
			fyne.Do(func() {
				prog.Hide()
				a.ShowMonitorScreen(envVars, cmdParts, visibleServices)
			})
		}()
		return
	}
	a.ShowMonitorScreen(envVars, cmdParts, visibleServices)
}
