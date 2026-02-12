package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type carbonioTheme struct{}

func NewCarbonioTheme() fyne.Theme {
	return &carbonioTheme{}
}

func (t *carbonioTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNamePrimary:
		return color.NRGBA{R: 125, G: 86, B: 244, A: 255} // #7D56F4
	case theme.ColorNameSuccess:
		return color.NRGBA{R: 0, G: 200, B: 0, A: 255}
	case theme.ColorNameWarning:
		return color.NRGBA{R: 255, G: 165, B: 0, A: 255}
	case theme.ColorNameError:
		return color.NRGBA{R: 255, G: 60, B: 60, A: 255}
	}
	return theme.DefaultTheme().Color(name, variant)
}

func (t *carbonioTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *carbonioTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *carbonioTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}
