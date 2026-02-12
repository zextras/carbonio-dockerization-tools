package gui

import (
	"carbonio-docker-cli/internal/graph"
	"carbonio-docker-cli/internal/parser"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
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
		container.NewPadded(container.NewGridWithColumns(2, backBtn, nextBtn)),
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
	dialog.ShowConfirm("Clean Database",
		"Do you want to start with a clean database?\n\n"+
			"This will remove the PostgreSQL volume, deleting all data\n"+
			"(files, tasks, docs, etc.). Recommended for fresh testing.",
		func(clean bool) {
			a.cleanDatabase = clean
			a.startMonitor(envVars, cmdParts)
		}, a.window)
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

func (a *App) showSaveConfigDialog(afterSave func()) {
	dialog.ShowConfirm("Save Configuration",
		"Do you want to save this configuration to a file?",
		func(wantsSave bool) {
			if !wantsSave {
				afterSave()
				return
			}
			fd := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
				if err != nil {
					dialog.ShowError(err, a.window)
					afterSave()
					return
				}
				if writer == nil {
					afterSave()
					return
				}
				writer.Close()
				path := writer.URI().Path()
				if saveErr := a.saveConfig(path); saveErr != nil {
					dialog.ShowError(saveErr, a.window)
				}
				afterSave()
			}, a.window)
			fd.SetFilter(storage.NewExtensionFileFilter([]string{".yaml", ".yml"}))
			fd.SetFileName("carbonio-config.yaml")
			fd.Show()
		}, a.window)
}
