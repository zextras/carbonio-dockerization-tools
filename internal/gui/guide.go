package gui

import (
	"image/color"
	"net/url"

	"carbonio-dockerization-tools/internal/preflight"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ShowGuideDialog displays a large scrollable setup guide modal with live check buttons.
func ShowGuideDialog(win fyne.Window) {
	titleLabel := widget.NewRichText(&widget.TextSegment{
		Text: "Setup Guide",
		Style: widget.RichTextStyle{
			SizeName:  theme.SizeNameSubHeadingText,
			TextStyle: fyne.TextStyle{Bold: true},
		},
	})

	step1 := buildStep1Section(win)
	step2 := buildStep2Section()
	step3 := buildStep3Section(win)

	scrollContent := container.NewVBox(
		step1,
		widget.NewSeparator(),
		step2,
		widget.NewSeparator(),
		step3,
	)
	scroll := container.NewVScroll(scrollContent)
	scroll.SetMinSize(fyne.NewSize(700, 450))

	closeBtn := widget.NewButton("Close", nil)
	closeBtn.Importance = widget.MediumImportance

	minWidth := canvas.NewRectangle(color.Transparent)
	minWidth.SetMinSize(fyne.NewSize(750, 0))

	bg := canvas.NewRectangle(theme.OverlayBackgroundColor())
	bg.CornerRadius = 8

	inner := container.NewVBox(
		minWidth,
		titleLabel,
		widget.NewSeparator(),
		scroll,
		widget.NewSeparator(),
		closeBtn,
	)

	card := container.NewStack(bg, container.NewPadded(inner))
	pop := widget.NewModalPopUp(card, win.Canvas())
	closeBtn.OnTapped = func() { pop.Hide() }
	pop.Show()
}

func copyableCommand(cmd string, win fyne.Window) fyne.CanvasObject {
	entry := widget.NewEntry()
	entry.SetText(cmd)
	entry.Disable()

	copyBtn := widget.NewButton("Copy", func() {
		win.Clipboard().SetContent(cmd)
	})
	copyBtn.Importance = widget.MediumImportance

	return container.NewBorder(nil, nil, nil, copyBtn, entry)
}

func newLink(text, rawURL string) *widget.Hyperlink {
	u, _ := url.Parse(rawURL)
	link := widget.NewHyperlink(text, u)
	return link
}

func buildStep1Section(win fyne.Window) fyne.CanvasObject {
	heading := widget.NewRichText(&widget.TextSegment{
		Text: "Step 1: Install Docker",
		Style: widget.RichTextStyle{
			SizeName:  theme.SizeNameText,
			TextStyle: fyne.TextStyle{Bold: true},
		},
	})

	intro := widget.NewLabel("Install Docker Desktop (macOS/Windows) or Docker Engine (Linux).\nOn macOS/Windows, make sure Docker Desktop is running before proceeding (it does not start automatically at boot by default).")
	intro.Wrapping = fyne.TextWrapWord

	desktopLink := newLink("Docker Desktop (macOS / Windows)", "https://www.docker.com/products/docker-desktop")
	engineLink := newLink("Docker Engine (Linux)", "https://docs.docker.com/engine/install/")

	requirement := widget.NewRichTextFromMarkdown(
		"**Minimum requirement:** Docker Compose v" + preflight.MinDockerComposeVersion + " or higher.\n\nVerify with:",
	)
	requirement.Wrapping = fyne.TextWrapWord

	verifyCmd := copyableCommand("docker compose version", win)

	statusLabel := widget.NewRichText(&widget.TextSegment{
		Text:  " ",
		Style: widget.RichTextStyle{},
	})

	checkBtn := widget.NewButton("Check Docker", nil)
	checkBtn.Importance = widget.MediumImportance
	checkBtn.OnTapped = func() {
		checkBtn.SetText("Checking...")
		checkBtn.Disable()
		go func() {
			err := preflight.CheckDockerComposeVersion()
			fyne.Do(func() {
				if err == nil {
					statusLabel.Segments = []widget.RichTextSegment{
						&widget.TextSegment{
							Text: "Passed",
							Style: widget.RichTextStyle{
								ColorName: theme.ColorNameSuccess,
								TextStyle: fyne.TextStyle{Bold: true},
							},
						},
					}
				} else {
					statusLabel.Segments = []widget.RichTextSegment{
						&widget.TextSegment{
							Text: err.Error(),
							Style: widget.RichTextStyle{
								ColorName: theme.ColorNameError,
							},
						},
					}
				}
				statusLabel.Refresh()
				checkBtn.SetText("Check Docker")
				checkBtn.Enable()
			})
		}()
	}

	return container.NewVBox(
		heading,
		intro,
		desktopLink,
		engineLink,
		requirement,
		verifyCmd,
		checkBtn,
		statusLabel,
	)
}

func buildStep2Section() fyne.CanvasObject {
	heading := widget.NewRichText(&widget.TextSegment{
		Text: "Step 2: Connect to VPN",
		Style: widget.RichTextStyle{
			SizeName:  theme.SizeNameText,
			TextStyle: fyne.TextStyle{Bold: true},
		},
	})

	body := widget.NewLabel("You must be connected to the Zextras VPN to reach the container registry.\n\nConnect using your VPN client before proceeding to the next step.")
	body.Wrapping = fyne.TextWrapWord

	return container.NewVBox(heading, body)
}

func buildStep3Section(win fyne.Window) fyne.CanvasObject {
	heading := widget.NewRichText(&widget.TextSegment{
		Text: "Step 3: Registry Login",
		Style: widget.RichTextStyle{
			SizeName:  theme.SizeNameText,
			TextStyle: fyne.TextStyle{Bold: true},
		},
	})

	subA := widget.NewRichText(&widget.TextSegment{
		Text:  "A) Get a token from Okta",
		Style: widget.RichTextStyle{TextStyle: fyne.TextStyle{Bold: true}},
	})

	registryLink := newLink("registry.dev.zextras.com/login", "https://registry.dev.zextras.com/login")

	steps := widget.NewLabel("1. Open the link above and sign in with Okta\n2. Click your profile icon → API Keys → Create New API Key\n3. Set a label and expiration, then copy the token")
	steps.Wrapping = fyne.TextWrapWord

	subB := widget.NewRichText(&widget.TextSegment{
		Text:  "B) Log in to the registry",
		Style: widget.RichTextStyle{TextStyle: fyne.TextStyle{Bold: true}},
	})

	loginInstructions := widget.NewLabel("Run the following command (replace with your token and email):")
	loginInstructions.Wrapping = fyne.TextWrapWord

	loginCmd := copyableCommand(
		`echo "your_token" | docker login registry.dev.zextras.com -u name.surname@zextras.com --password-stdin`,
		win,
	)

	statusLabel := widget.NewRichText(&widget.TextSegment{
		Text:  " ",
		Style: widget.RichTextStyle{},
	})

	checkBtn := widget.NewButton("Check Registry", nil)
	checkBtn.Importance = widget.MediumImportance
	checkBtn.OnTapped = func() {
		checkBtn.SetText("Checking...")
		checkBtn.Disable()
		go func() {
			err := preflight.CheckRegistryConnectivity()
			fyne.Do(func() {
				if err == nil {
					statusLabel.Segments = []widget.RichTextSegment{
						&widget.TextSegment{
							Text: "Passed",
							Style: widget.RichTextStyle{
								ColorName: theme.ColorNameSuccess,
								TextStyle: fyne.TextStyle{Bold: true},
							},
						},
					}
				} else {
					statusLabel.Segments = []widget.RichTextSegment{
						&widget.TextSegment{
							Text: err.Error(),
							Style: widget.RichTextStyle{
								ColorName: theme.ColorNameError,
							},
						},
					}
				}
				statusLabel.Refresh()
				checkBtn.SetText("Check Registry")
				checkBtn.Enable()
			})
		}()
	}

	return container.NewVBox(
		heading,
		subA,
		registryLink,
		steps,
		subB,
		loginInstructions,
		loginCmd,
		checkBtn,
		statusLabel,
	)
}
