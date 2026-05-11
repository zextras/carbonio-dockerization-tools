// SPDX-FileCopyrightText: 2026 Zextras <https://www.zextras.com>
//
// SPDX-License-Identifier: AGPL-3.0-only

package gui

import (
	_ "embed"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

//go:embed carbonio-docker-logo.png
var logoPNG []byte

func logoResource() fyne.Resource {
	return fyne.NewStaticResource("carbonio-docker-logo.png", logoPNG)
}

func LogoResource() fyne.Resource {
	return logoResource()
}

func newLogo(size float32) *canvas.Image {
	img := canvas.NewImageFromResource(logoResource())
	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(fyne.NewSize(size, size))
	return img
}
