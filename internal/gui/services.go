package gui

import (
	"carbonio-docker-cli/internal/config"
	"carbonio-docker-cli/internal/graph"
	"carbonio-docker-cli/internal/parser"
	"fmt"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type serviceItem struct {
	name        string
	imageBase   string
	defaultTag  string
	customImage string
	customTag   string
	selected    bool
	isBackend   bool
	isRequired  bool
	tagLocked   bool
	useCustom   bool
	deps        []string
}

func extractImageBase(imageURL string) string {
	if imageURL == "" {
		return ""
	}
	lastColon := strings.LastIndex(imageURL, ":")
	lastSlash := strings.LastIndex(imageURL, "/")
	if lastColon > lastSlash && lastColon != -1 {
		return imageURL[:lastColon]
	}
	return imageURL
}

func newSectionHeader(text string) *widget.RichText {
	return widget.NewRichText(&widget.TextSegment{
		Text: text,
		Style: widget.RichTextStyle{
			SizeName:  theme.SizeNameSubHeadingText,
			ColorName: theme.ColorNamePrimary,
			TextStyle: fyne.TextStyle{Bold: true},
		},
	})
}

func newCompactLabel(text string, bold bool) *widget.RichText {
	style := widget.RichTextStyle{SizeName: theme.SizeNameCaptionText}
	if bold {
		style.TextStyle = fyne.TextStyle{Bold: true}
	}
	return widget.NewRichText(&widget.TextSegment{Text: text, Style: style})
}

func setCompactLabelStyle(label *widget.RichText, text string, italic bool, bold bool) {
	label.Segments = []widget.RichTextSegment{
		&widget.TextSegment{
			Text: text,
			Style: widget.RichTextStyle{
				SizeName:  theme.SizeNameCaptionText,
				TextStyle: fyne.TextStyle{Italic: italic, Bold: bold},
			},
		},
	}
	label.Refresh()
}

func (a *App) ShowServicesScreen(resolver *graph.DependencyResolver) {
	backendItems := buildBackendItems(a.parsedConfig, a.edition)
	frontendItems := buildFrontendItems(a.parsedConfig)

	backendSection := newSectionHeader("Backend Services")
	backendList := buildServiceList(backendItems, resolver, a.window)

	frontendSection := newSectionHeader("Composed UI")
	frontendList := buildServiceList(frontendItems, nil, a.window)

	startBtn := widget.NewButton("Start", func() {
		backend, frontend := collectSelections(backendItems, frontendItems, a.edition)
		a.pendingBackend = backend
		a.pendingFrontend = frontend

		a.showSaveConfigDialog(func() {
			envVars, cmdParts, err := a.buildDockerCommand()
			if err != nil {
				dialog.ShowError(err, a.window)
				return
			}
			a.promptCleanDatabaseThenMonitor(envVars, cmdParts)
		})
	})
	startBtn.Importance = widget.HighImportance

	exportBtn := widget.NewButton("Export Config", func() {
		backend, frontend := collectSelections(backendItems, frontendItems, a.edition)
		a.pendingBackend = backend
		a.pendingFrontend = frontend

		fd := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
			if err != nil {
				dialog.ShowError(err, a.window)
				return
			}
			if writer == nil {
				return
			}
			writer.Close()
			path := writer.URI().Path()
			if saveErr := a.saveConfig(path); saveErr != nil {
				dialog.ShowError(saveErr, a.window)
				return
			}
			showSuccessDialog("Config Exported", fmt.Sprintf("Saved to:\n%s", path), a.window)
		}, a.window)
		fd.SetFileName("carbonio-config.yaml")
		fd.Show()
	})

	backBtn := widget.NewButton("Back", func() {
		a.ShowEditionScreen()
	})

	editionLabel := "CE (Community Edition)"
	if a.edition == parser.EditionAdvanced {
		editionLabel = "Advanced"
	}

	logo := newLogo(32)
	title := widget.NewRichTextFromMarkdown("# Configure Services")
	titleRow := container.NewHBox(logo, title)
	editionSubtitle := widget.NewRichText(&widget.TextSegment{
		Text: editionLabel,
		Style: widget.RichTextStyle{
			SizeName:  theme.SizeNameSubHeadingText,
			TextStyle: fyne.TextStyle{Bold: true},
		},
	})
	description := widget.NewRichText(&widget.TextSegment{
		Text: "Select and configure the services to deploy.",
		Style: widget.RichTextStyle{
			SizeName:  theme.SizeNameCaptionText,
			ColorName: theme.ColorNamePlaceHolder,
		},
	})

	scrollContent := container.NewVBox(
		backendSection,
		backendList,
		widget.NewSeparator(),
		frontendSection,
		frontendList,
	)

	topSection := container.NewPadded(container.NewPadded(container.NewVBox(titleRow, editionSubtitle, description, widget.NewSeparator())))
	bottomSection := container.NewPadded(container.NewPadded(container.NewHBox(
		wideButton(backBtn, 120),
		layout.NewSpacer(),
		wideButton(exportBtn, 150),
		wideButton(startBtn, 120),
	)))

	content := container.NewBorder(
		topSection,
		bottomSection,
		nil, nil,
		container.NewVScroll(container.NewPadded(scrollContent)),
	)

	a.window.SetContent(content)
}

func buildBackendItems(parsedConfig *parser.ParsedConfig, edition parser.Edition) []*serviceItem {
	var required, optional []*serviceItem

	names := make([]string, 0, len(parsedConfig.BackendServices))
	for name := range parsedConfig.BackendServices {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		svc := parsedConfig.BackendServices[name]
		if parser.GlobalDockerConfig.IsServiceHidden(name) {
			continue
		}
		imageBase := extractImageBase(svc.DefaultImage)
		if imageBase == "" {
			imageBase = name
		}
		tagLocked := parser.GlobalDockerConfig.IsTagLocked(name, svc.DefaultTag, true)

		item := &serviceItem{
			name:       name,
			imageBase:  imageBase,
			defaultTag: svc.DefaultTag,
			selected:   true,
			isBackend:  true,
			isRequired: svc.IsRequired,
			tagLocked:  tagLocked,
			deps:       svc.DependsOn,
		}

		if item.isRequired {
			required = append(required, item)
		} else {
			optional = append(optional, item)
		}
	}

	return append(required, optional...)
}

func buildFrontendItems(parsedConfig *parser.ParsedConfig) []*serviceItem {
	var required, optional []*serviceItem

	names := make([]string, 0, len(parsedConfig.FrontendImages))
	for name := range parsedConfig.FrontendImages {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		ui := parsedConfig.FrontendImages[name]
		imageBase := extractImageBase(ui.DefaultImage)
		tagLocked := parser.GlobalDockerConfig.IsTagLocked(name, ui.DefaultTag, false)

		item := &serviceItem{
			name:       name,
			imageBase:  imageBase,
			defaultTag: ui.DefaultTag,
			selected:   true,
			isBackend:  false,
			isRequired: ui.IsProxy,
			tagLocked:  tagLocked,
		}

		if item.isRequired {
			required = append(required, item)
		} else {
			optional = append(optional, item)
		}
	}

	return append(required, optional...)
}

func buildServiceList(items []*serviceItem, resolver *graph.DependencyResolver, win fyne.Window) *fyne.Container {
	rows := container.NewVBox()

	for _, item := range items {
		item := item // capture

		check := widget.NewCheck("", func(checked bool) {
			item.selected = checked
			if checked && item.isBackend && resolver != nil {
				autoSelectDeps(item, items, resolver)
			}
		})
		check.SetChecked(item.selected)
		if item.isRequired {
			check.SetChecked(true)
			check.Disable()
		}

		nameLabel := newCompactLabel(item.name, item.isRequired)

		tag := item.defaultTag
		if item.customTag != "" {
			tag = item.customTag
		}
		tagSelect := widget.NewSelect([]string{tag}, func(selected string) {
			if selected != item.defaultTag {
				item.customTag = selected
			} else {
				item.customTag = ""
			}
		})
		tagSelect.SetSelected(tag)
		externalImage := !IsOurRegistry(item.imageBase)
		if item.tagLocked || item.useCustom || externalImage {
			tagSelect.Disable()
		}

		customLabel := widget.NewRichText(&widget.TextSegment{
			Text: "(custom)",
			Style: widget.RichTextStyle{
				SizeName:  theme.SizeNameCaptionText,
				ColorName: theme.ColorNamePlaceHolder,
				TextStyle: fyne.TextStyle{Italic: true},
			},
		})
		if !item.useCustom {
			customLabel.Hide()
		}

		var resetBtn *widget.Button
		resetBtn = widget.NewButton("Reset", func() {
			item.customImage = ""
			item.customTag = ""
			item.useCustom = false

			setCompactLabelStyle(nameLabel, item.name, false, item.isRequired)
			customLabel.Hide()
			resetBtn.Hide()

			if !item.tagLocked {
				tagSelect.Enable()
			}
			tagSelect.SetSelected(item.defaultTag)
		})
		if !item.useCustom {
			resetBtn.Hide()
		}

		customBtn := widget.NewButton("Custom", func() {
			showCustomDialog(item, tagSelect, nameLabel, customLabel, resetBtn, win)
		})
		if item.tagLocked || externalImage {
			customBtn.Disable()
		}

		leftCol := container.NewHBox(check, nameLabel, customLabel)
		rightCol := container.NewHBox(tagSelect, customBtn, resetBtn)
		row := container.NewGridWithColumns(3, leftCol, rightCol, layout.NewSpacer())
		rows.Add(row)

		// Fetch tags in background
		go func(item *serviceItem, sel *widget.Select) {
			tags := FetchTags(item.imageBase)
			if tags == nil {
				return
			}
			sel.Options = tags
			// Auto-select best default: devel > latest > first
			bestTag := tags[0]
			for _, t := range tags {
				if t == "devel" {
					bestTag = "devel"
					break
				}
				if t == "latest" {
					bestTag = "latest"
				}
			}
			if item.customTag == "" {
				sel.SetSelected(bestTag)
				item.defaultTag = bestTag
			}
			sel.Refresh()
		}(item, tagSelect)
	}

	return rows
}

func showCustomDialog(item *serviceItem, tagSelect *widget.Select, nameLabel *widget.RichText, customLabel *widget.RichText, resetBtn *widget.Button, win fyne.Window) {
	currentImage := item.imageBase
	if item.customImage != "" {
		currentImage = item.customImage
	}
	currentTag := item.defaultTag
	if item.customTag != "" {
		currentTag = item.customTag
	}

	originalValue := fmt.Sprintf("%s:%s", currentImage, currentTag)

	imageEntry := widget.NewEntry()
	imageEntry.SetText(originalValue)
	imageEntry.SetPlaceHolder("registry.example.com/namespace/image:tag")

	applyBtn := widget.NewButton("Apply", nil)
	applyBtn.Importance = widget.HighImportance
	applyBtn.Disable()

	cancelBtn := widget.NewButton("Cancel", nil)

	imageEntry.OnChanged = func(s string) {
		trimmed := strings.TrimSpace(s)
		if trimmed != originalValue && trimmed != "" {
			applyBtn.Enable()
		} else {
			applyBtn.Disable()
		}
	}

	entryLabel := widget.NewRichText(&widget.TextSegment{
		Text: "Full image URL:",
		Style: widget.RichTextStyle{
			SizeName:  theme.SizeNameCaptionText,
			ColorName: theme.ColorNamePlaceHolder,
		},
	})

	dialogTitle := widget.NewRichText(&widget.TextSegment{
		Text: fmt.Sprintf("Custom Image — %s", item.name),
		Style: widget.RichTextStyle{
			SizeName:  theme.SizeNameSubHeadingText,
			TextStyle: fyne.TextStyle{Bold: true},
		},
	})

	content := container.NewVBox(
		dialogTitle,
		widget.NewSeparator(),
		entryLabel,
		imageEntry,
		widget.NewSeparator(),
		container.NewGridWithColumns(2, cancelBtn, applyBtn),
	)

	d := dialog.NewCustomWithoutButtons("", content, win)

	applyBtn.OnTapped = func() {
		fullURL := strings.TrimSpace(imageEntry.Text)
		if fullURL == "" {
			return
		}

		var parsedImage, parsedTag string
		lastColon := strings.LastIndex(fullURL, ":")
		lastSlash := strings.LastIndex(fullURL, "/")
		if lastColon > lastSlash && lastColon != -1 {
			parsedImage = fullURL[:lastColon]
			parsedTag = fullURL[lastColon+1:]
		} else {
			parsedImage = fullURL
			parsedTag = item.defaultTag
		}

		if parsedImage == item.imageBase {
			// Same image, different tag — just a tag change, not a custom image
			// Validate the tag against fetched registry tags if available
			if len(tagSelect.Options) > 1 {
				tagFound := false
				for _, opt := range tagSelect.Options {
					if opt == parsedTag {
						tagFound = true
						break
					}
				}
				if !tagFound {
					dialog.ShowError(
						fmt.Errorf("Tag \"%s\" not found in registry for %s.\n\nAvailable tags can be seen in the dropdown.", parsedTag, item.name),
						win,
					)
					return
				}
			}

			item.customImage = ""
			item.customTag = parsedTag
			item.useCustom = false

			setCompactLabelStyle(nameLabel, item.name, false, item.isRequired)
			customLabel.Hide()
			resetBtn.Hide()

			if !item.tagLocked {
				tagSelect.Enable()
			}
			tagSelect.SetSelected(parsedTag)
		} else {
			// Different image — true custom
			item.customImage = parsedImage
			item.customTag = parsedTag
			item.useCustom = true

			setCompactLabelStyle(nameLabel, item.name, true, false)
			customLabel.Show()
			resetBtn.Show()
			tagSelect.Disable()
		}

		d.Hide()
	}

	cancelBtn.OnTapped = func() {
		d.Hide()
	}

	d.Resize(fyne.NewSize(650, 250))
	d.Show()
}

func autoSelectDeps(item *serviceItem, allItems []*serviceItem, resolver *graph.DependencyResolver) {
	deps := resolver.ResolveDependencies(item.name)
	for _, depName := range deps {
		for _, other := range allItems {
			if other.name == depName {
				other.selected = true
			}
		}
	}
}

func collectSelections(backendItems, frontendItems []*serviceItem, edition parser.Edition) (map[string]*config.ImageConfig, map[string]*config.ImageConfig) {
	backend := make(map[string]*config.ImageConfig)
	frontend := make(map[string]*config.ImageConfig)

	for _, item := range backendItems {
		if !item.selected {
			continue
		}
		image := item.imageBase
		tag := item.defaultTag
		if item.useCustom {
			image = item.customImage
			tag = item.customTag
		} else {
			if item.customImage != "" {
				image = item.customImage
			}
			if item.customTag != "" {
				tag = item.customTag
			}
		}
		backend[item.name] = &config.ImageConfig{Image: image, Tag: tag}
	}

	for _, item := range frontendItems {
		image := item.imageBase
		tag := item.defaultTag
		if item.useCustom {
			image = item.customImage
			tag = item.customTag
		} else {
			if item.customImage != "" {
				image = item.customImage
			}
			if item.customTag != "" {
				tag = item.customTag
			}
		}
		if !item.selected {
			tag = "disabled"
		}
		frontend[item.name] = &config.ImageConfig{Image: image, Tag: tag}
	}

	return backend, frontend
}
