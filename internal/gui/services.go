// SPDX-FileCopyrightText: 2026 Zextras <https://www.zextras.com>
//
// SPDX-License-Identifier: AGPL-3.0-only

package gui

import (
	"carbonio-dockerization-tools/internal/config"
	"carbonio-dockerization-tools/internal/graph"
	"carbonio-dockerization-tools/internal/parser"
	"fmt"
	"image/color"
	"log"
	"os"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
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
	if err := CheckRegistryAuth(); err != nil {
		ShowFatalErrorDialog(
			fmt.Sprintf("Cannot fetch image tags from the registry.\n\n%v", err),
			a.window, func() { ShowGuideDialog(a.window) }, a.logPath,
		)
		return
	}

	backendItems := buildBackendItems(a.parsedConfig, a.edition)
	frontendItems := buildFrontendItems(a.parsedConfig)

	// Probe: verify we can actually list tags (auth may pass but tag API may be down)
	if probeImage := findProbeImage(backendItems); probeImage != "" {
		if _, err := FetchTags(probeImage); err != nil {
			ShowFatalErrorDialog(
				fmt.Sprintf("Registry authentication succeeded, but cannot fetch image tags.\n\n%v", err),
				a.window, func() { ShowGuideDialog(a.window) }, a.logPath,
			)
			return
		}
	}

	backendSection := newSectionHeader("Backend Services")
	backendResult := buildServiceList(backendItems, resolver, a.window)

	frontendSection := newSectionHeader("Composed UI")
	frontendResult := buildServiceList(frontendItems, nil, a.window)

	// Toolbar: Select All, Deselect All, Hide uncustomizable
	selectAllBtn := widget.NewButton("Select All", func() {
		for _, item := range backendItems {
			if !item.selected && !item.isRequired {
				item.selected = true
				if chk, ok := backendResult.checks[item.name]; ok {
					chk.SetChecked(true)
				}
			}
		}
		for _, item := range frontendItems {
			if !item.selected && !item.isRequired {
				item.selected = true
				if chk, ok := frontendResult.checks[item.name]; ok {
					chk.SetChecked(true)
				}
			}
		}
	})

	deselectAllBtn := widget.NewButton("Deselect All", func() {
		for _, item := range backendItems {
			if item.selected && !item.isRequired {
				item.selected = false
				if chk, ok := backendResult.checks[item.name]; ok {
					chk.SetChecked(false)
				}
			}
		}
		for _, item := range frontendItems {
			if item.selected && !item.isRequired {
				item.selected = false
				if chk, ok := frontendResult.checks[item.name]; ok {
					chk.SetChecked(false)
				}
			}
		}
	})

	allItems := append(backendItems, frontendItems...)
	allRows := make(map[string]*fyne.Container)
	for k, v := range backendResult.rows {
		allRows[k] = v
	}
	for k, v := range frontendResult.rows {
		allRows[k] = v
	}

	hideCheck := widget.NewCheck("Hide uncustomizable", func(checked bool) {
		for _, item := range allItems {
			if isUncustomizable(item) {
				if row, ok := allRows[item.name]; ok {
					if checked {
						row.Hide()
					} else {
						row.Show()
					}
				}
			}
		}
	})
	hideCheck.SetChecked(true)
	// Apply initial hide
	for _, item := range allItems {
		if isUncustomizable(item) {
			if row, ok := allRows[item.name]; ok {
				row.Hide()
			}
		}
	}

	toolbarContent := container.NewHBox(
		selectAllBtn,
		deselectAllBtn,
		layout.NewSpacer(),
		hideCheck,
	)
	toolbarBorder := canvas.NewRectangle(color.Transparent)
	toolbarBorder.StrokeColor = theme.PrimaryColor()
	toolbarBorder.StrokeWidth = 1.5
	toolbarBorder.CornerRadius = 6
	padH := canvas.NewRectangle(color.Transparent)
	padH.SetMinSize(fyne.NewSize(8, 0))
	padV := canvas.NewRectangle(color.Transparent)
	padV.SetMinSize(fyne.NewSize(0, 6))
	toolbarInner := container.NewBorder(padV, padV, padH, padH, toolbarContent)
	toolbarBox := container.NewStack(toolbarBorder, toolbarInner)
	toolbar := container.NewGridWithColumns(2, toolbarBox, layout.NewSpacer())

	copyBtn := widget.NewButton("Copy startup command", func() {
		backend, frontend := collectSelections(backendItems, frontendItems, a.edition)
		a.pendingBackend = backend
		a.pendingFrontend = frontend

		result, err := a.buildDockerCommand()
		if err != nil {
			showErrorDialog(err.Error(), a.window)
			return
		}
		a.window.Clipboard().SetContent(result.StartupCommand())
		showSuccessDialog("Copied", "Startup command copied to clipboard.", a.window)
	})

	startBtn := widget.NewButton("Start", func() {
		backend, frontend := collectSelections(backendItems, frontendItems, a.edition)
		a.pendingBackend = backend
		a.pendingFrontend = frontend

		result, err := a.buildDockerCommand()
		if err != nil {
			showErrorDialog(err.Error(), a.window)
			return
		}
		a.promptCleanPersistenceThenMonitor(result)
	})
	startBtn.Importance = widget.HighImportance

	exportBtn := widget.NewButton("Export Config", func() {
		backend, frontend := collectSelections(backendItems, frontendItems, a.edition)
		a.pendingBackend = backend
		a.pendingFrontend = frontend

		fd := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
			if err != nil {
				showErrorDialog(err.Error(), a.window)
				return
			}
			if writer == nil {
				return
			}
			writerPath := writer.URI().Path()
			writer.Close()
			os.Remove(writerPath)
			path := writerPath + ".carbonio-dockerization"
			if saveErr := a.saveConfig(path); saveErr != nil {
				showErrorDialog(saveErr.Error(), a.window)
				return
			}
			showSuccessDialog("Config Exported", fmt.Sprintf("Saved to:\n%s", path), a.window)
		}, a.window)
		fd.SetFileName("carbonio-config")
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
		backendResult.container,
		widget.NewSeparator(),
		frontendSection,
		frontendResult.container,
	)

	topSection := container.NewPadded(container.NewPadded(container.NewVBox(a.versionInfoLabel(), titleRow, editionSubtitle, description, toolbar, widget.NewSeparator())))
	bottomSection := container.NewPadded(container.NewPadded(container.NewHBox(
		wideButton(backBtn, 120),
		wideButton(exportBtn, 150),
		layout.NewSpacer(),
		wideButton(copyBtn, 160),
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

type serviceListResult struct {
	container *fyne.Container
	checks    map[string]*widget.Check
	rows      map[string]*fyne.Container
}

func isUncustomizable(item *serviceItem) bool {
	if !item.isRequired {
		return false
	}
	externalImage := !IsOurRegistry(item.imageBase)
	return item.tagLocked || externalImage
}

func buildServiceList(items []*serviceItem, resolver *graph.DependencyResolver, win fyne.Window) *serviceListResult {
	rowsContainer := container.NewVBox()
	checks := make(map[string]*widget.Check)
	rowMap := make(map[string]*fyne.Container)

	for _, item := range items {
		item := item // capture

		check := widget.NewCheck("", nil)
		checks[item.name] = check
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
		rowMap[item.name] = row
		rowsContainer.Add(row)

		// Fetch tags in background
		go func(item *serviceItem, sel *widget.Select) {
			tags, err := FetchTags(item.imageBase)
			if err != nil {
				log.Printf("FetchTags: %v", err)
				return
			}
			if tags == nil {
				return
			}
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
			fyne.Do(func() {
				sel.Options = tags
				if item.customTag == "" {
					sel.SetSelected(bestTag)
					item.defaultTag = bestTag
				}
				sel.Refresh()
			})
		}(item, tagSelect)
	}

	// Wire up OnChanged handlers now that all checks are in the map
	for _, item := range items {
		item := item // capture
		chk := checks[item.name]
		chk.OnChanged = func(checked bool) {
			item.selected = checked
			if item.isBackend && resolver != nil {
				propagateDependencyChange(item, checked, items, checks, resolver)
			}
		}
	}

	return &serviceListResult{
		container: rowsContainer,
		checks:    checks,
		rows:      rowMap,
	}
}

// propagateDependencyChange handles bidirectional dependency propagation:
// - On select: auto-select all transitive dependencies
// - On deselect: auto-deselect all services that transitively depend on this one
func propagateDependencyChange(item *serviceItem, checked bool, allItems []*serviceItem, checks map[string]*widget.Check, resolver *graph.DependencyResolver) {
	if checked {
		// Select all dependencies (what this service needs)
		deps := resolver.ResolveDependencies(item.name)
		for _, depName := range deps {
			for _, other := range allItems {
				if other.name == depName && !other.selected {
					other.selected = true
					if chk, ok := checks[other.name]; ok {
						chk.SetChecked(true)
					}
				}
			}
		}
	} else {
		// Deselect all dependents (services that need this one)
		dependents := resolver.ResolveDependents(item.name)
		for _, depName := range dependents {
			for _, other := range allItems {
				if other.name == depName && other.selected && !other.isRequired {
					other.selected = false
					if chk, ok := checks[other.name]; ok {
						chk.SetChecked(false)
					}
				}
			}
		}
	}
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

	minWidth := canvas.NewRectangle(color.Transparent)
	minWidth.SetMinSize(fyne.NewSize(550, 0))

	buttonSpacer := canvas.NewRectangle(color.Transparent)
	buttonSpacer.SetMinSize(fyne.NewSize(0, 4))

	inner := container.NewVBox(
		minWidth,
		dialogTitle,
		widget.NewSeparator(),
		entryLabel,
		imageEntry,
		buttonSpacer,
		widget.NewSeparator(),
		container.NewGridWithColumns(2, cancelBtn, applyBtn),
	)

	bg := canvas.NewRectangle(theme.OverlayBackgroundColor())
	bg.CornerRadius = 8
	card := container.NewStack(bg, container.NewPadded(inner))
	pop := widget.NewModalPopUp(card, win.Canvas())

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
					showErrorDialog(
						fmt.Sprintf("Tag \"%s\" not found in registry for %s.\n\nAvailable tags can be seen in the dropdown.", parsedTag, item.name),
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

		pop.Hide()
	}

	cancelBtn.OnTapped = func() {
		pop.Hide()
	}

	pop.Show()
}

// findProbeImage returns the imageBase of the first service that belongs to our registry,
// to use as a probe for verifying tag-list API availability.
func findProbeImage(items []*serviceItem) string {
	for _, item := range items {
		if IsOurRegistry(item.imageBase) {
			return item.imageBase
		}
	}
	return ""
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
