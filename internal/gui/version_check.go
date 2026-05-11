// SPDX-FileCopyrightText: 2026 Zextras <https://www.zextras.com>
//
// SPDX-License-Identifier: AGPL-3.0-only

package gui

import (
	"context"
	"encoding/json"
	"fmt"
	"image/color"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type semver struct {
	major, minor, patch int
}

func parseSemver(s string) (semver, bool) {
	s = strings.TrimPrefix(s, "v")
	parts := strings.SplitN(s, ".", 3)
	if len(parts) != 3 {
		return semver{}, false
	}
	major, err1 := strconv.Atoi(parts[0])
	minor, err2 := strconv.Atoi(parts[1])
	patch, err3 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil || err3 != nil {
		return semver{}, false
	}
	return semver{major, minor, patch}, true
}

func (a semver) newerThan(b semver) bool {
	if a.major != b.major {
		return a.major > b.major
	}
	if a.minor != b.minor {
		return a.minor > b.minor
	}
	return a.patch > b.patch
}

const latestReleaseURL = "https://api.github.com/repos/zextras/carbonio-dockerization-tools/releases/latest"

// checkForUpdate queries GitHub Releases API and returns the latest release
// tag if it is newer than appVersion. Returns ("", false) if no update is
// available, the version is "dev", or any error occurs.
func checkForUpdate(appVersion string) (latestTag string, hasUpdate bool) {
	if appVersion == "" || appVersion == "dev" {
		return "", false
	}

	currentVer, ok := parseSemver(appVersion)
	if !ok {
		log.Printf("version_check: cannot parse current version %q", appVersion)
		return "", false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, latestReleaseURL, nil)
	if err != nil {
		log.Printf("version_check: failed to create request: %v", err)
		return "", false
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("version_check: GitHub API request failed: %v", err)
		return "", false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("version_check: GitHub API returned status %d", resp.StatusCode)
		return "", false
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		log.Printf("version_check: failed to parse response: %v", err)
		return "", false
	}

	if release.TagName == "" {
		return "", false
	}

	latestVer, ok := parseSemver(release.TagName)
	if !ok {
		log.Printf("version_check: cannot parse latest version %q", release.TagName)
		return "", false
	}

	if latestVer.newerThan(currentVer) {
		return release.TagName, true
	}
	return "", false
}

func showUpdateAvailableDialog(latestVersion, currentVersion string, win fyne.Window) {
	titleLabel := widget.NewRichText(&widget.TextSegment{
		Text: "Update Available",
		Style: widget.RichTextStyle{
			SizeName:  theme.SizeNameSubHeadingText,
			TextStyle: fyne.TextStyle{Bold: true},
		},
	})
	messageLabel := widget.NewLabel(fmt.Sprintf(
		"A newer version (%s) is available. You are running %s.\nConsider updating for the latest features and fixes.",
		latestVersion, currentVersion,
	))
	messageLabel.Wrapping = fyne.TextWrapWord

	ignoreBtn := widget.NewButton("Ignore", nil)
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
		container.NewGridWithColumns(2, ignoreBtn, okBtn),
	)

	card := container.NewStack(bg, container.NewPadded(inner))
	pop := widget.NewModalPopUp(card, win.Canvas())

	ignoreBtn.OnTapped = func() { pop.Hide() }
	okBtn.OnTapped = func() { pop.Hide() }

	pop.Show()
}

// CheckForUpdate runs a background version check against GitHub and shows
// a non-blocking modal if a newer version is available. Errors are silently
// logged without affecting the user.
func (a *App) CheckForUpdate() {
	go func() {
		latestTag, hasUpdate := checkForUpdate(a.appVersion)
		if hasUpdate {
			fyne.Do(func() {
				showUpdateAvailableDialog(latestTag, a.appVersion, a.window)
			})
		}
	}()
}
