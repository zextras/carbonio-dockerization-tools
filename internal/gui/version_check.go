package gui

import (
	"context"
	"fmt"
	"image/color"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var semverTagRegex = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)$`)

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

// checkForUpdate queries GitHub for tags via git ls-remote and returns the
// latest semver tag if it is newer than appVersion. Returns ("", false) if
// no update is available, the version is "dev", or any error occurs.
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

	cmd := exec.CommandContext(ctx, "git", "ls-remote", "--tags", "https://github.com/zextras/carbonio-dockerization-tools.git")
	out, err := cmd.Output()
	if err != nil {
		log.Printf("version_check: git ls-remote failed: %v", err)
		return "", false
	}

	var best semver
	var bestTag string

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		ref := parts[1]
		// Skip dereferenced tags (^{})
		if strings.HasSuffix(ref, "^{}") {
			continue
		}
		tag := strings.TrimPrefix(ref, "refs/tags/")
		if !semverTagRegex.MatchString(tag) {
			continue
		}
		ver, ok := parseSemver(tag)
		if !ok {
			continue
		}
		if ver.newerThan(best) {
			best = ver
			bestTag = tag
		}
	}

	if bestTag == "" {
		return "", false
	}

	if best.newerThan(currentVer) {
		return bestTag, true
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
