package gui

import (
	"carbonio-docker-cli/internal/graph"
	"carbonio-docker-cli/internal/parser"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
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
			dialog.ShowError(err, a.window)
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

func (a *App) promptCleanDatabaseThenMonitor(envVars string, cmdParts []string) {
	titleLabel := widget.NewRichText(&widget.TextSegment{
		Text: "Clean Database",
		Style: widget.RichTextStyle{
			SizeName:  theme.SizeNameSubHeadingText,
			TextStyle: fyne.TextStyle{Bold: true},
		},
	})
	messageLabel := widget.NewLabel(
		"Do you want to start with a clean database?\n\n" +
			"This will remove the PostgreSQL volume, deleting all data\n" +
			"(files, tasks, docs, etc.). Recommended for fresh testing.")

	yesBtn := widget.NewButton("Yes, clean", nil)
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
		a.cleanDatabase = true
		a.startMonitor(envVars, cmdParts)
	}
	noBtn.OnTapped = func() {
		pop.Hide()
		a.cleanDatabase = false
		a.startMonitor(envVars, cmdParts)
	}

	pop.Show()
}

func (a *App) handleConfigImport(filePath string) {
	err := a.loadConfigFromFile(filePath)
	if err != nil {
		dialog.ShowError(err, a.window)
		return
	}

	envVars, cmdParts, err := a.buildDockerCommand()
	if err != nil {
		dialog.ShowError(err, a.window)
		return
	}

	a.promptCleanDatabaseThenMonitor(envVars, cmdParts)
}

func (a *App) startMonitor(envVars string, cmdParts []string) {
	if a.cleanDatabase {
		a.executor.CleanDatabaseVolumes()
	}
	visibleServices := a.buildVisibleServicesList()
	a.ShowMonitorScreen(envVars, cmdParts, visibleServices)
}
