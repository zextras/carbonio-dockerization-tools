// SPDX-FileCopyrightText: 2026 Zextras <https://www.zextras.com>
//
// SPDX-License-Identifier: AGPL-3.0-only

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
	case "stateReady":
		return color.NRGBA{R: 255, G: 220, B: 50, A: 255}
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
	switch name {
	case theme.SizeNamePadding:
		return 3
	case theme.SizeNameInnerPadding:
		return 3
	case theme.SizeNameText:
		return 11
	}
	return theme.DefaultTheme().Size(name)
}
