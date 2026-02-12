package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var (
	colorRunning  = color.NRGBA{R: 0, G: 200, B: 0, A: 255}
	colorStarting = color.NRGBA{R: 255, G: 165, B: 0, A: 255}
	colorError    = color.NRGBA{R: 255, G: 60, B: 60, A: 255}
	colorUnknown  = color.NRGBA{R: 150, G: 150, B: 150, A: 255}
	colorStopped  = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
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

func NewStatusBadge(state string) *fyne.Container {
	var c color.Color
	switch state {
	case "Running":
		c = colorRunning
	case "Starting", "Pulling", "Creating":
		c = colorStarting
	case "Error":
		c = colorError
	default:
		c = colorUnknown
	}

	dot := canvas.NewCircle(c)
	dot.Resize(fyne.NewSize(12, 12))
	dotContainer := container.NewWithoutLayout(dot)
	dotContainer.Resize(fyne.NewSize(16, 16))

	label := widget.NewLabel(state)

	return container.NewHBox(dotContainer, label)
}
