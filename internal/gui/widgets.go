package gui

import (
	"fmt"
	"image/color"
	"io"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func wideButton(btn *widget.Button, minWidth float32) fyne.CanvasObject {
	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(minWidth, 36))
	return container.NewStack(spacer, btn)
}

func showSuccessDialog(title, message string, win fyne.Window) {
	titleLabel := widget.NewRichText(&widget.TextSegment{
		Text: title,
		Style: widget.RichTextStyle{
			SizeName:  theme.SizeNameSubHeadingText,
			TextStyle: fyne.TextStyle{Bold: true},
		},
	})
	messageLabel := widget.NewLabel(message)

	okBtn := widget.NewButton("OK", nil)
	okBtn.Importance = widget.HighImportance

	minWidth := canvas.NewRectangle(color.Transparent)
	minWidth.SetMinSize(fyne.NewSize(500, 0))

	bg := canvas.NewRectangle(theme.OverlayBackgroundColor())
	bg.CornerRadius = 8

	inner := container.NewVBox(
		minWidth,
		titleLabel,
		widget.NewSeparator(),
		messageLabel,
		widget.NewSeparator(),
		okBtn,
	)

	card := container.NewStack(bg, container.NewPadded(inner))
	pop := widget.NewModalPopUp(card, win.Canvas())
	okBtn.OnTapped = func() { pop.Hide() }
	pop.Show()
}

func showProgressModal(title, message string, win fyne.Window) *widget.PopUp {
	titleLabel := widget.NewRichText(&widget.TextSegment{
		Text: title,
		Style: widget.RichTextStyle{
			SizeName:  theme.SizeNameSubHeadingText,
			TextStyle: fyne.TextStyle{Bold: true},
		},
	})
	messageLabel := widget.NewLabel(message)

	progress := widget.NewProgressBarInfinite()

	minWidth := canvas.NewRectangle(color.Transparent)
	minWidth.SetMinSize(fyne.NewSize(500, 0))

	bg := canvas.NewRectangle(theme.OverlayBackgroundColor())
	bg.CornerRadius = 8

	inner := container.NewVBox(
		minWidth,
		titleLabel,
		widget.NewSeparator(),
		messageLabel,
		progress,
	)

	card := container.NewStack(bg, container.NewPadded(inner))
	pop := widget.NewModalPopUp(card, win.Canvas())
	pop.Show()
	return pop
}

func showCleanupCompleteDialog(win fyne.Window, onOK func()) {
	titleLabel := widget.NewRichText(&widget.TextSegment{
		Text: "Cleanup Complete",
		Style: widget.RichTextStyle{
			SizeName:  theme.SizeNameSubHeadingText,
			TextStyle: fyne.TextStyle{Bold: true},
		},
	})
	messageLabel := widget.NewLabel("All containers have been stopped and cleaned up.")

	okBtn := widget.NewButton("OK", nil)
	okBtn.Importance = widget.HighImportance

	minWidth := canvas.NewRectangle(color.Transparent)
	minWidth.SetMinSize(fyne.NewSize(500, 0))

	bg := canvas.NewRectangle(theme.OverlayBackgroundColor())
	bg.CornerRadius = 8

	inner := container.NewVBox(
		minWidth,
		titleLabel,
		widget.NewSeparator(),
		messageLabel,
		widget.NewSeparator(),
		okBtn,
	)

	card := container.NewStack(bg, container.NewPadded(inner))
	pop := widget.NewModalPopUp(card, win.Canvas())
	okBtn.OnTapped = func() {
		pop.Hide()
		if onOK != nil {
			onOK()
		}
	}
	pop.Show()
}

func showErrorDialog(message string, win fyne.Window) {
	titleLabel := widget.NewRichText(&widget.TextSegment{
		Text: "Error",
		Style: widget.RichTextStyle{
			SizeName:  theme.SizeNameSubHeadingText,
			TextStyle: fyne.TextStyle{Bold: true},
		},
	})
	messageLabel := widget.NewLabel(message)
	messageLabel.Wrapping = fyne.TextWrapWord

	okBtn := widget.NewButton("OK", nil)
	okBtn.Importance = widget.MediumImportance

	minWidth := canvas.NewRectangle(color.Transparent)
	minWidth.SetMinSize(fyne.NewSize(500, 0))

	bg := canvas.NewRectangle(theme.OverlayBackgroundColor())
	bg.CornerRadius = 8

	inner := container.NewVBox(
		minWidth,
		titleLabel,
		widget.NewSeparator(),
		messageLabel,
		widget.NewSeparator(),
		okBtn,
	)

	card := container.NewStack(bg, container.NewPadded(inner))
	pop := widget.NewModalPopUp(card, win.Canvas())
	okBtn.OnTapped = func() { pop.Hide() }
	pop.Show()
}

// ShowFatalErrorDialog shows a custom error modal. OK closes the window (exits the app).
// When onGuide is non-nil, a "Guide" button is shown alongside "OK".
// When logPath is non-empty, an "Export Logs" button lets the user save the log before exiting.
func ShowFatalErrorDialog(message string, win fyne.Window, onGuide func(), logPath string) {
	titleLabel := widget.NewRichText(&widget.TextSegment{
		Text: "Error",
		Style: widget.RichTextStyle{
			SizeName:  theme.SizeNameSubHeadingText,
			TextStyle: fyne.TextStyle{Bold: true},
		},
	})
	messageLabel := widget.NewLabel(message)
	messageLabel.Wrapping = fyne.TextWrapWord

	okBtn := widget.NewButton("OK", nil)
	okBtn.Importance = widget.MediumImportance

	var buttonList []fyne.CanvasObject
	if onGuide != nil {
		guideBtn := widget.NewButton("Guide", nil)
		guideBtn.Importance = widget.MediumImportance
		guideBtn.OnTapped = func() { onGuide() }
		buttonList = append(buttonList, guideBtn)
	}

	var exportBtn *widget.Button
	if logPath != "" {
		exportBtn = widget.NewButton("Export Logs", nil)
		exportBtn.Importance = widget.MediumImportance
		buttonList = append(buttonList, exportBtn)
	}

	buttonList = append(buttonList, okBtn)
	buttons := container.NewGridWithColumns(len(buttonList), buttonList...)

	minWidth := canvas.NewRectangle(color.Transparent)
	minWidth.SetMinSize(fyne.NewSize(500, 0))

	bg := canvas.NewRectangle(theme.OverlayBackgroundColor())
	bg.CornerRadius = 8

	inner := container.NewVBox(
		minWidth,
		titleLabel,
		widget.NewSeparator(),
		messageLabel,
		widget.NewSeparator(),
		buttons,
	)

	card := container.NewStack(bg, container.NewPadded(inner))
	pop := widget.NewModalPopUp(card, win.Canvas())
	okBtn.OnTapped = func() {
		pop.Hide()
		win.Close()
	}
	if exportBtn != nil {
		exportBtn.OnTapped = func() {
			pop.Hide()
			exportLogsThenClose(logPath, win)
		}
	}
	pop.Show()
}

// exportLogsThenClose opens a save dialog for the log file, then closes the app.
func exportLogsThenClose(logPath string, win fyne.Window) {
	fd := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil || writer == nil {
			win.Close()
			return
		}
		defer writer.Close()

		src, err := os.Open(logPath)
		if err != nil {
			showErrorDialog(fmt.Sprintf("Failed to open log file: %v", err), win)
			win.Close()
			return
		}
		defer src.Close()

		if _, err := io.Copy(writer, src); err != nil {
			showErrorDialog(fmt.Sprintf("Failed to copy logs: %v", err), win)
			win.Close()
			return
		}

		destPath := writer.URI().Path()
		showSuccessDialogThenClose("Logs Exported", fmt.Sprintf("Saved to:\n%s", destPath), win)
	}, win)

	timestamp := time.Now().Format("2006-01-02_150405")
	fd.SetFileName(fmt.Sprintf("carbonio-dockerization-gui-logs-%s.log", timestamp))
	fd.Show()
}

func showSuccessDialogThenClose(title, message string, win fyne.Window) {
	titleLabel := widget.NewRichText(&widget.TextSegment{
		Text: title,
		Style: widget.RichTextStyle{
			SizeName:  theme.SizeNameSubHeadingText,
			TextStyle: fyne.TextStyle{Bold: true},
		},
	})
	messageLabel := widget.NewLabel(message)

	okBtn := widget.NewButton("OK", nil)
	okBtn.Importance = widget.HighImportance

	minW := canvas.NewRectangle(color.Transparent)
	minW.SetMinSize(fyne.NewSize(500, 0))

	bg := canvas.NewRectangle(theme.OverlayBackgroundColor())
	bg.CornerRadius = 8

	inner := container.NewVBox(
		minW,
		titleLabel,
		widget.NewSeparator(),
		messageLabel,
		widget.NewSeparator(),
		okBtn,
	)

	card := container.NewStack(bg, container.NewPadded(inner))
	pop := widget.NewModalPopUp(card, win.Canvas())
	okBtn.OnTapped = func() {
		pop.Hide()
		win.Close()
	}
	pop.Show()
}

// ShowVPNRetryDialog shows a custom modal for registry connectivity failure.
// Retry closes the modal and calls onRetry. Close exits the app.
// When onGuide is non-nil, a "Guide" button is shown alongside Close and Retry.
func ShowVPNRetryDialog(message string, win fyne.Window, onRetry func(), onGuide func()) {
	titleLabel := widget.NewRichText(&widget.TextSegment{
		Text: "Connection Error",
		Style: widget.RichTextStyle{
			SizeName:  theme.SizeNameSubHeadingText,
			TextStyle: fyne.TextStyle{Bold: true},
		},
	})
	messageLabel := widget.NewLabel(message)
	messageLabel.Wrapping = fyne.TextWrapWord

	retryBtn := widget.NewButton("Retry", nil)
	retryBtn.Importance = widget.HighImportance

	closeBtn := widget.NewButton("Close", nil)
	closeBtn.Importance = widget.MediumImportance

	var buttons fyne.CanvasObject
	if onGuide != nil {
		guideBtn := widget.NewButton("Guide", nil)
		guideBtn.Importance = widget.MediumImportance
		guideBtn.OnTapped = func() { onGuide() }
		buttons = container.NewGridWithColumns(3, closeBtn, guideBtn, retryBtn)
	} else {
		buttons = container.NewGridWithColumns(2, closeBtn, retryBtn)
	}

	minW := canvas.NewRectangle(color.Transparent)
	minW.SetMinSize(fyne.NewSize(500, 0))

	bg := canvas.NewRectangle(theme.OverlayBackgroundColor())
	bg.CornerRadius = 8

	inner := container.NewVBox(
		minW,
		titleLabel,
		widget.NewSeparator(),
		messageLabel,
		widget.NewSeparator(),
		buttons,
	)

	card := container.NewStack(bg, container.NewPadded(inner))
	pop := widget.NewModalPopUp(card, win.Canvas())

	retryBtn.OnTapped = func() {
		pop.Hide()
		if onRetry != nil {
			onRetry()
		}
	}

	closeBtn.OnTapped = func() {
		pop.Hide()
		win.Close()
	}

	pop.Show()
}

// ShowLoadingScreen sets the window content to a centered loading screen with logo,
// title, progress bar and a step label. Returns a function to update the step text.
func ShowLoadingScreen(win fyne.Window) func(step string) {
	logo := newLogo(96)
	title := widget.NewRichTextFromMarkdown("# Carbonio Docker GUI")

	stepLabel := widget.NewRichText(&widget.TextSegment{
		Text: "Initializing...",
		Style: widget.RichTextStyle{
			SizeName:  theme.SizeNameCaptionText,
			ColorName: theme.ColorNamePlaceHolder,
		},
	})

	progress := widget.NewProgressBarInfinite()

	content := container.NewVBox(
		container.NewCenter(logo),
		title,
		container.NewPadded(container.NewVBox(
			stepLabel,
			progress,
		)),
	)

	win.SetContent(container.NewCenter(content))

	return func(step string) {
		stepLabel.Segments = []widget.RichTextSegment{
			&widget.TextSegment{
				Text: step,
				Style: widget.RichTextStyle{
					SizeName:  theme.SizeNameCaptionText,
					ColorName: theme.ColorNamePlaceHolder,
				},
			},
		}
		stepLabel.Refresh()
	}
}

func showVersionMismatchDialog(configVersion, appVersion string, win fyne.Window, onContinue func(), onGoBack func()) {
	titleLabel := widget.NewRichText(&widget.TextSegment{
		Text: "Version Mismatch",
		Style: widget.RichTextStyle{
			SizeName:  theme.SizeNameSubHeadingText,
			TextStyle: fyne.TextStyle{Bold: true},
		},
	})
	messageLabel := widget.NewLabel(fmt.Sprintf(
		"This configuration was created with version %s, but you are running version %s. It may not be fully compatible.",
		configVersion, appVersion,
	))
	messageLabel.Wrapping = fyne.TextWrapWord

	goBackBtn := widget.NewButton("Go Back", nil)
	continueBtn := widget.NewButton("Continue", nil)
	continueBtn.Importance = widget.HighImportance

	minWidth := canvas.NewRectangle(color.Transparent)
	minWidth.SetMinSize(fyne.NewSize(500, 0))

	bg := canvas.NewRectangle(theme.OverlayBackgroundColor())
	bg.CornerRadius = 8

	inner := container.NewVBox(
		minWidth,
		titleLabel,
		widget.NewSeparator(),
		messageLabel,
		widget.NewSeparator(),
		container.NewGridWithColumns(2, goBackBtn, continueBtn),
	)

	card := container.NewStack(bg, container.NewPadded(inner))
	pop := widget.NewModalPopUp(card, win.Canvas())

	goBackBtn.OnTapped = func() {
		pop.Hide()
		if onGoBack != nil {
			onGoBack()
		}
	}
	continueBtn.OnTapped = func() {
		pop.Hide()
		if onContinue != nil {
			onContinue()
		}
	}

	pop.Show()
}
