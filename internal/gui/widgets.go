package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

var (
	colorRunning  = color.NRGBA{R: 0, G: 200, B: 0, A: 255}
	colorStarting = color.NRGBA{R: 255, G: 165, B: 0, A: 255}
	colorError    = color.NRGBA{R: 255, G: 60, B: 60, A: 255}
	colorUnknown  = color.NRGBA{R: 150, G: 150, B: 150, A: 255}
)

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
