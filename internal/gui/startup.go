package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
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

	logo := newLogo(96)
	title := widget.NewRichTextFromMarkdown("# Carbonio Docker GUI")

	modeLabel := widget.NewRichText(&widget.TextSegment{
		Text: "Select startup mode:",
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
		wideButton(nextBtn, 200),
	)

	a.window.SetContent(container.NewCenter(content))
}

func (a *App) showFileOpen() {
	fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			showErrorDialog(err.Error(), a.window)
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
