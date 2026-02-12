package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

func (a *App) ShowStartupScreen() {
	choice := widget.NewRadioGroup([]string{
		"Import configuration from file",
		"Custom configuration",
	}, nil)
	choice.SetSelected("Custom configuration")

	nextBtn := widget.NewButton("Next", func() {
		if choice.Selected == "Import configuration from file" {
			a.showFileOpen()
		} else {
			a.ShowEditionScreen()
		}
	})
	nextBtn.Importance = widget.HighImportance

	title := widget.NewRichTextFromMarkdown("# Carbonio Docker GUI")

	content := container.NewVBox(
		title,
		widget.NewLabel("Select startup mode:"),
		choice,
		widget.NewSeparator(),
		nextBtn,
	)

	a.window.SetContent(container.NewCenter(content))
}

func (a *App) showFileOpen() {
	fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, a.window)
			return
		}
		if reader == nil {
			return // cancelled
		}
		reader.Close()
		path := reader.URI().Path()
		a.handleConfigImport(path)
	}, a.window)
	fd.SetFilter(storage.NewExtensionFileFilter([]string{".yaml", ".yml"}))
	fd.Show()
}
