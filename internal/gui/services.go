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

func (a *App) ShowServicesScreen(resolver *graph.DependencyResolver) {
	backendItems := buildBackendItems(a.parsedConfig, a.edition)
	frontendItems := buildFrontendItems(a.parsedConfig)

	backendSection := widget.NewRichTextFromMarkdown("### Backend Services")
	backendList := buildServiceList(backendItems, resolver, a.window)

	frontendSection := widget.NewRichTextFromMarkdown("### Composed UI")
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
			}
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

	title := widget.NewRichTextFromMarkdown("# Select Services and UI Images")
	subtitle := widget.NewLabel(fmt.Sprintf("Edition: %s", editionLabel))

	scrollContent := container.NewVBox(
		backendSection,
		backendList,
		widget.NewSeparator(),
		frontendSection,
		frontendList,
	)

	content := container.NewBorder(
		container.NewVBox(title, subtitle, widget.NewSeparator()),
		container.NewVBox(widget.NewSeparator(), container.NewHBox(backBtn, layout.NewSpacer(), exportBtn, startBtn)),
		nil, nil,
		container.NewVScroll(scrollContent),
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

		tag := item.defaultTag
		if item.customTag != "" {
			tag = item.customTag
		}
		img := item.imageBase
		if item.customImage != "" {
			img = item.customImage
		}

		nameLabel := widget.NewLabel(item.name)
		nameLabel.TextStyle = fyne.TextStyle{Bold: item.isRequired}

		imageLabel := widget.NewLabel(fmt.Sprintf("%s:%s", img, tag))

		editBtn := widget.NewButton("Edit", func() {
			showEditDialog(item, imageLabel, win)
		})
		if item.tagLocked {
			editBtn.Disable()
		}

		row := container.NewHBox(check, nameLabel, layout.NewSpacer(), imageLabel, editBtn)
		rows.Add(row)
	}

	return rows
}

func showEditDialog(item *serviceItem, imageLabel *widget.Label, win fyne.Window) {
	currentImage := item.imageBase
	if item.customImage != "" {
		currentImage = item.customImage
	}
	currentTag := item.defaultTag
	if item.customTag != "" {
		currentTag = item.customTag
	}

	imageEntry := widget.NewEntry()
	imageEntry.SetText(currentImage)
	tagEntry := widget.NewEntry()
	tagEntry.SetText(currentTag)

	items := []*widget.FormItem{
		widget.NewFormItem("Image", imageEntry),
		widget.NewFormItem("Tag", tagEntry),
	}

	dialog.ShowForm("Edit Image", "Apply", "Cancel", items, func(confirmed bool) {
		if !confirmed {
			return
		}
		newImage := strings.TrimSpace(imageEntry.Text)
		newTag := strings.TrimSpace(tagEntry.Text)
		if newImage != "" && newImage != item.imageBase {
			item.customImage = newImage
		}
		if newTag != "" && newTag != item.defaultTag {
			item.customTag = newTag
		}

		img := item.imageBase
		if item.customImage != "" {
			img = item.customImage
		}
		tag := item.defaultTag
		if item.customTag != "" {
			tag = item.customTag
		}
		imageLabel.SetText(fmt.Sprintf("%s:%s", img, tag))
	}, win)
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
		if item.customImage != "" {
			image = item.customImage
		}
		tag := item.defaultTag
		if item.customTag != "" {
			tag = item.customTag
		}
		backend[item.name] = &config.ImageConfig{Image: image, Tag: tag}
	}

	for _, item := range frontendItems {
		image := item.imageBase
		if item.customImage != "" {
			image = item.customImage
		}
		tag := item.defaultTag
		if item.customTag != "" {
			tag = item.customTag
		}
		if !item.selected {
			tag = "disabled"
		}
		frontend[item.name] = &config.ImageConfig{Image: image, Tag: tag}
	}

	return backend, frontend
}
