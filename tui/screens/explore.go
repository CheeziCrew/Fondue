package screens

import (
	"fmt"
	"image/color"
	"io"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/CheeziCrew/curd"
	"github.com/CheeziCrew/fondue/scanner"
	"github.com/sahilm/fuzzy"
)

// BackToMenuMsg signals the root model to return to the menu.
type BackToMenuMsg struct{}

// ExportGraphMsg requests the root model to generate a subgraph image.
type ExportGraphMsg struct {
	Service string
	Hops    int
}

// GraphExportedMsg is the result of a graph export attempt.
type GraphExportedMsg struct {
	Path string
	Err  error
}

// ── Explore Model ───────────────────────────────────────────────────

type exploreView int

const (
	exploreList exploreView = iota
	exploreDetail
)

// ExploreModel is the interactive dependency graph explorer.
type ExploreModel struct {
	services        []scanner.Service
	nameIdx         *scanner.NameIndex
	list            list.Model
	view            exploreView
	selectedService *scanner.Service
	detailCursor    int
	navStack        []navEntry
	width           int
	height          int

	// Graph export hop input
	enteringHops bool
	hopInput     string
	exportFlash  string
}

type navEntry struct {
	service *scanner.Service
	cursor  int
}

func NewExplore(services []scanner.Service, idx *scanner.NameIndex, width, height int) ExploreModel {
	m := ExploreModel{
		services: services,
		nameIdx:  idx,
		view:     exploreList,
		width:    width,
		height:   height,
	}
	m.list = newServiceList(services, idx, width, height)
	return m
}

func (m ExploreModel) Init() tea.Cmd { return nil }

func (m ExploreModel) Update(msg tea.Msg) (ExploreModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "q":
			if m.list.FilterState() != list.Filtering {
				return m, func() tea.Msg { return BackToMenuMsg{} }
			}
		case "esc":
			if m.view == exploreDetail {
				if !m.popNav() {
					m.view = exploreList
					m.detailCursor = 0
				}
				return m, nil
			}
			if m.list.FilterState() != list.Filtering {
				return m, func() tea.Msg { return BackToMenuMsg{} }
			}
		}
	}

	switch m.view {
	case exploreList:
		return handleExploreListUpdate(m, msg)
	case exploreDetail:
		return handleExploreDetailUpdate(m, msg)
	}

	return m, nil
}

func (m ExploreModel) View() string {
	switch m.view {
	case exploreDetail:
		return renderDetail(m)
	default:
		return m.list.View()
	}
}

func (m *ExploreModel) pushNav() {
	m.navStack = append(m.navStack, navEntry{
		service: m.selectedService,
		cursor:  m.detailCursor,
	})
}

func (m *ExploreModel) popNav() bool {
	if len(m.navStack) == 0 {
		return false
	}
	last := m.navStack[len(m.navStack)-1]
	m.navStack = m.navStack[:len(m.navStack)-1]
	m.selectedService = last.service
	m.detailCursor = last.cursor
	return true
}

// ── List view ───────────────────────────────────────────────────────

type serviceItem struct {
	service scanner.Service
}

func (i serviceItem) FilterValue() string {
	parts := []string{i.service.Name}
	for _, integ := range i.service.Integrations {
		parts = append(parts, integ.ClientID)
	}
	for _, dep := range i.service.DependedOnBy {
		parts = append(parts, dep)
	}
	return strings.Join(parts, " ")
}

type serviceDelegate struct {
	nameIdx *scanner.NameIndex
}

func (d serviceDelegate) Height() int                             { return 3 }
func (d serviceDelegate) Spacing() int                            { return 1 }
func (d serviceDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d serviceDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(serviceItem)
	if !ok {
		return
	}

	svc := item.service
	out := len(svc.Integrations)
	in := len(svc.DependedOnBy)
	selected := index == m.Index()
	width := m.Width()

	var nameStyle, outBadge, inBadge, isoBadge lipgloss.Style
	var border lipgloss.Style

	if selected {
		nameStyle = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
		outBadge = lipgloss.NewStyle().Foreground(colorBg).Background(colorYellow).Bold(true).Padding(0, 1)
		inBadge = lipgloss.NewStyle().Foreground(colorBg).Background(colorGreen).Bold(true).Padding(0, 1)
		isoBadge = lipgloss.NewStyle().Foreground(colorDim).Italic(true)
		border = lipgloss.NewStyle().
			BorderLeft(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(colorAccent).
			PaddingLeft(1)
	} else {
		nameStyle = lipgloss.NewStyle().Foreground(colorFg)
		outBadge = lipgloss.NewStyle().Foreground(colorYellow).Bold(true)
		inBadge = lipgloss.NewStyle().Foreground(colorGreen).Bold(true)
		isoBadge = lipgloss.NewStyle().Foreground(colorDim).Italic(true)
		border = lipgloss.NewStyle().
			BorderLeft(true).
			BorderStyle(lipgloss.HiddenBorder()).
			PaddingLeft(1)
	}

	name := nameStyle.Render(svc.Name)

	var badges string
	if out > 0 {
		if selected {
			badges += "  " + outBadge.Render(fmt.Sprintf("↑ %d out", out))
		} else {
			badges += "  " + outBadge.Render(fmt.Sprintf("↑ %d", out))
		}
	}
	if in > 0 {
		if selected {
			badges += "  " + inBadge.Render(fmt.Sprintf("↓ %d in", in))
		} else {
			badges += "  " + inBadge.Render(fmt.Sprintf("↓ %d", in))
		}
	}
	if out == 0 && in == 0 {
		badges = "  " + isoBadge.Render("○ isolated")
	}

	staleCount := scanner.StaleCount(&svc, allServices(m), d.nameIdx)
	if staleCount > 0 {
		staleBadge := lipgloss.NewStyle().Foreground(colorYellow).Bold(true)
		if selected {
			staleBadge = lipgloss.NewStyle().Foreground(colorBg).Background(colorYellow).Bold(true).Padding(0, 1)
		}
		badges += "  " + staleBadge.Render(fmt.Sprintf("⚠ %d stale", staleCount))
	}

	line1 := name + badges

	const maxPreview = 70
	var line2 string
	if out > 0 {
		arrow := lipgloss.NewStyle().Foreground(colorYellow).Render("→")
		depColor := colorFg
		if !selected {
			depColor = colorDim
		}
		line2 = arrow + " " + renderDepList(svc.Integrations, nil, depColor, maxPreview)
	}

	var line3 string
	if in > 0 {
		arrow := lipgloss.NewStyle().Foreground(colorGreen).Render("←")
		depColor := colorFg
		if !selected {
			depColor = colorDim
		}
		line3 = arrow + " " + renderDepList(nil, svc.DependedOnBy, depColor, maxPreview)
	}

	lines := line1
	if line2 != "" {
		lines += "\n" + line2
	}
	if line3 != "" {
		lines += "\n" + line3
	}

	lineCount := 1
	if line2 != "" {
		lineCount++
	}
	if line3 != "" {
		lineCount++
	}
	for lineCount < 3 {
		lines += "\n"
		lineCount++
	}

	rendered := border.Width(width - 4).Render(lines)
	fmt.Fprint(w, rendered)
}

func renderDepList(integrations []scanner.Integration, names []string, c color.Color, maxWidth int) string {
	depStyle := lipgloss.NewStyle().Foreground(c)
	sep := dimStyle.Render(" · ")

	var rawNames []string
	if integrations != nil {
		for _, integ := range integrations {
			rawNames = append(rawNames, integ.ClientID)
		}
	} else {
		rawNames = names
	}

	var result string
	for i, n := range rawNames {
		styled := depStyle.Render(n)
		candidate := result
		if i > 0 {
			candidate += sep
		}
		candidate += styled

		if lipgloss.Width(candidate) > maxWidth {
			result += dimStyle.Render("...")
			break
		}
		if i > 0 {
			result += sep
		}
		result += styled
	}
	return result
}

func allServices(m list.Model) []scanner.Service {
	items := m.Items()
	services := make([]scanner.Service, len(items))
	for i, item := range items {
		if si, ok := item.(serviceItem); ok {
			services[i] = si.service
		}
	}
	return services
}

// ── Fuzzy filter ────────────────────────────────────────────────────

func fuzzyFilter(term string, targets []string) []list.Rank {
	matches := fuzzy.Find(term, targets)
	var ranks []list.Rank
	for _, m := range matches {
		ranks = append(ranks, list.Rank{
			Index:          m.Index,
			MatchedIndexes: m.MatchedIndexes,
		})
	}
	return ranks
}

func newServiceList(services []scanner.Service, idx *scanner.NameIndex, width, height int) list.Model {
	items := make([]list.Item, len(services))
	for i, s := range services {
		items[i] = serviceItem{service: s}
	}

	l := list.New(items, serviceDelegate{nameIdx: idx}, width, height)
	l.Filter = fuzzyFilter
	l.Title = "  Fondue"
	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorAccent).
		Padding(0, 2).
		MarginBottom(1)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.StatusBar = lipgloss.NewStyle().Foreground(colorDim).PaddingLeft(2)

	return l
}

func handleExploreListUpdate(m ExploreModel, msg tea.Msg) (ExploreModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}
		switch msg.String() {
		case "enter":
			if item, ok := m.list.SelectedItem().(serviceItem); ok {
				m.selectedService = &item.service
				m.view = exploreDetail
				m.navStack = nil
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// ── Detail view ─────────────────────────────────────────────────────

func renderDetail(m ExploreModel) string {
	svc := m.selectedService
	if svc == nil {
		return ""
	}

	contentWidth := m.width - 8
	if contentWidth < 40 {
		contentWidth = 40
	}

	var content strings.Builder

	// Header
	title := svc.Name
	if svc.Version != "" {
		title += " · " + svc.Version
	}
	titleLine := detailTitleStyle.Render(fmt.Sprintf("  %s", title))
	content.WriteString(titleLine + "\n")
	content.WriteString(detailPathStyle.Render(svc.Path) + "\n")

	// Stats bar
	outCount := len(svc.Integrations)
	inCount := len(svc.DependedOnBy)
	statsLine := "  " +
		badgeOutStyle.Render(fmt.Sprintf(" %d outbound", outCount)) +
		"  " +
		badgeInStyle.Render(fmt.Sprintf(" %d inbound", inCount))
	content.WriteString(statsLine + "\n")

	// Depends on
	content.WriteString(
		sectionHeaderStyle.Render(
			arrowOutStyle.Render("")+"  Depends on "+
				sectionCountStyle.Render(fmt.Sprintf("(%d)", outCount)),
		) + "\n",
	)

	if outCount == 0 {
		content.WriteString(noneStyle.Render("  no outbound dependencies") + "\n")
	} else {
		navigable := getNavigableIntegrations(svc.Integrations, m.nameIdx)
		for _, integ := range svc.Integrations {
			isInternal := m.nameIdx.IsInternal(integ.ClientID)

			navIdx := -1
			for ni, nInteg := range navigable {
				if nInteg.ClientID == integ.ClientID {
					navIdx = ni
					break
				}
			}
			isCursor := navIdx >= 0 && navIdx == m.detailCursor

			prefix := "    "
			var line string

			if isCursor {
				prefix = "  " + cursorStyle.Render("") + " "
				line = cursorStyle.Render(integ.ClientID)
			} else if isInternal {
				prefix = "  " + arrowOutStyle.Render("") + " "
				line = internalStyle.Render(integ.ClientID)
			} else {
				prefix = "  " + dimStyle.Render("") + " "
				line = externalStyle.Render(integ.ClientID)
			}

			if !isInternal {
				line += " " + externalTagStyle.Render("ext")
			}

			if isInternal {
				line += renderVersionAnnotation(integ, m.services, m.nameIdx)
			}

			content.WriteString(prefix + line + "\n")
		}
	}

	// Depended on by
	content.WriteString(
		sectionHeaderStyle.Render(
			arrowInStyle.Render("")+"  Depended on by "+
				sectionCountStyle.Render(fmt.Sprintf("(%d)", inCount)),
		) + "\n",
	)

	if inCount == 0 {
		content.WriteString(noneStyle.Render("  no inbound dependencies") + "\n")
	} else {
		navigable := getNavigableIntegrations(svc.Integrations, m.nameIdx)
		navOffset := len(navigable)

		for i, dep := range svc.DependedOnBy {
			isCursor := m.detailCursor == navOffset+i

			var prefix, line string
			if isCursor {
				prefix = "  " + cursorStyle.Render("") + " "
				line = cursorStyle.Render(dep)
			} else {
				prefix = "  " + arrowInStyle.Render("") + " "
				line = internalStyle.Render(dep)
			}

			line += renderReverseStaleAnnotation(dep, svc, m.services, m.nameIdx)
			content.WriteString(prefix + line + "\n")
		}
	}

	box := detailBoxStyle.Width(contentWidth).Render(content.String())

	var footer string
	if m.enteringHops {
		prompt := lipgloss.NewStyle().Bold(true).Foreground(colorAccent).Render("  Hops: ")
		val := lipgloss.NewStyle().Bold(true).Foreground(colorFg).Render(m.hopInput)
		cursor := lipgloss.NewStyle().Foreground(colorAccent).Render("█")
		footer = prompt + val + cursor + "  " + dimStyle.Render("enter to export · esc to cancel")
	} else {
		backDesc := "back"
		if len(m.navStack) > 0 {
			backDesc = "back (history)"
		}
		hints := []curd.Hint{
			{Key: "esc", Desc: backDesc},
			{Key: "enter", Desc: "navigate"},
			{Key: "j/k", Desc: "move"},
			{Key: "g", Desc: "graph"},
			{Key: "q", Desc: "menu"},
		}
		footer = curd.RenderHintBar(st, hints)
		if m.exportFlash != "" {
			flashStyle := lipgloss.NewStyle().Foreground(colorGreen).Bold(true).PaddingLeft(2)
			if strings.HasPrefix(m.exportFlash, "✗") {
				flashStyle = lipgloss.NewStyle().Foreground(colorRed).Bold(true).PaddingLeft(2)
			}
			footer = flashStyle.Render(m.exportFlash) + "\n" + footer
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, box, footer)
}

func renderVersionAnnotation(integ scanner.Integration, services []scanner.Service, idx *scanner.NameIndex) string {
	if integ.SpecVersion == "" {
		return ""
	}

	targetName := idx.Resolve(integ.ClientID)
	if targetName == "" {
		return ""
	}
	target := scanner.FindService(targetName, services)
	if target == nil || target.Version == "" {
		return ""
	}

	if integ.SpecVersion == target.Version {
		return " " + versionMatchStyle.Render(integ.SpecVersion+" ✓")
	}

	return " " + staleStyle.Render(integ.SpecVersion+" → "+target.Version+" STALE")
}

func renderReverseStaleAnnotation(depName string, svc *scanner.Service, services []scanner.Service, idx *scanner.NameIndex) string {
	if svc.Version == "" {
		return ""
	}

	dependent := scanner.FindService(depName, services)
	if dependent == nil {
		return ""
	}

	for _, integ := range dependent.Integrations {
		if integ.SpecVersion == "" {
			continue
		}
		target := idx.Resolve(integ.ClientID)
		if target != svc.Name {
			continue
		}
		if integ.SpecVersion == svc.Version {
			return " " + versionMatchStyle.Render(integ.SpecVersion + " ✓")
		}
		return " " + staleStyle.Render(integ.SpecVersion + " → " + svc.Version + " STALE")
	}
	return ""
}

func getNavigableIntegrations(integrations []scanner.Integration, idx *scanner.NameIndex) []scanner.Integration {
	var nav []scanner.Integration
	for _, integ := range integrations {
		if idx.IsInternal(integ.ClientID) {
			nav = append(nav, integ)
		}
	}
	return nav
}

func getNavigableCount(svc *scanner.Service, idx *scanner.NameIndex) int {
	return len(getNavigableIntegrations(svc.Integrations, idx)) + len(svc.DependedOnBy)
}

func handleExploreDetailUpdate(m ExploreModel, msg tea.Msg) (ExploreModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		// Handle hop input mode
		if m.enteringHops {
			switch msg.String() {
			case "enter":
				hops := 1
				if m.hopInput != "" {
					if n, err := strconv.Atoi(m.hopInput); err == nil && n > 0 {
						hops = n
					}
				}
				m.enteringHops = false
				m.hopInput = ""
				return m, func() tea.Msg {
					return ExportGraphMsg{
						Service: m.selectedService.Name,
						Hops:    hops,
					}
				}
			case "esc":
				m.enteringHops = false
				m.hopInput = ""
				return m, nil
			case "backspace":
				if len(m.hopInput) > 0 {
					m.hopInput = m.hopInput[:len(m.hopInput)-1]
				}
				return m, nil
			default:
				ch := msg.String()
				if len(ch) == 1 && ch[0] >= '0' && ch[0] <= '9' {
					m.hopInput += ch
				}
				return m, nil
			}
		}

		maxItems := getNavigableCount(m.selectedService, m.nameIdx)

		switch msg.String() {
		case "backspace":
			if !m.popNav() {
				m.view = exploreList
				m.detailCursor = 0
			}
			return m, nil

		case "up", "k":
			if m.detailCursor > 0 {
				m.detailCursor--
			}
			return m, nil

		case "down", "j":
			if m.detailCursor < maxItems-1 {
				m.detailCursor++
			}
			return m, nil

		case "g":
			m.enteringHops = true
			m.hopInput = "1"
			m.exportFlash = ""
			return m, nil

		case "enter":
			targetName := resolveDetailTarget(m)
			if targetName != "" {
				for i := range m.services {
					if m.services[i].Name == targetName {
						m.pushNav()
						m.selectedService = &m.services[i]
						m.detailCursor = 0
						return m, nil
					}
				}
			}
			return m, nil
		}

	case GraphExportedMsg:
		if msg.Err != nil {
			m.exportFlash = "✗ " + msg.Err.Error()
		} else {
			m.exportFlash = "✓ opened graph"
		}
		return m, nil
	}
	return m, nil
}

func resolveDetailTarget(m ExploreModel) string {
	svc := m.selectedService
	navigable := getNavigableIntegrations(svc.Integrations, m.nameIdx)
	navOffset := len(navigable)

	if m.detailCursor < navOffset {
		return m.nameIdx.Resolve(navigable[m.detailCursor].ClientID)
	}

	idx := m.detailCursor - navOffset
	if idx < len(svc.DependedOnBy) {
		return svc.DependedOnBy[idx]
	}
	return ""
}
