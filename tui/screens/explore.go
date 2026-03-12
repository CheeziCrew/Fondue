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
		if m, cmd, handled := m.handleGlobalKey(msg); handled {
			return m, cmd
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

func (m ExploreModel) handleGlobalKey(msg tea.KeyPressMsg) (ExploreModel, tea.Cmd, bool) {
	switch msg.String() {
	case "q":
		if m.list.FilterState() != list.Filtering {
			return m, func() tea.Msg { return BackToMenuMsg{} }, true
		}
	case "esc":
		return m.handleEscKey()
	}
	return m, nil, false
}

func (m ExploreModel) handleEscKey() (ExploreModel, tea.Cmd, bool) {
	if m.view == exploreDetail {
		if !m.popNav() {
			m.view = exploreList
			m.detailCursor = 0
		}
		return m, nil, true
	}
	if m.list.FilterState() != list.Filtering {
		return m, func() tea.Msg { return BackToMenuMsg{} }, true
	}
	return m, nil, false
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

type delegateStyles struct {
	name, outBadge, inBadge, isoBadge, border lipgloss.Style
}

func selectedDelegateStyles() delegateStyles {
	return delegateStyles{
		name:     lipgloss.NewStyle().Bold(true).Foreground(colorAccent),
		outBadge: lipgloss.NewStyle().Foreground(colorBg).Background(colorYellow).Bold(true).Padding(0, 1),
		inBadge:  lipgloss.NewStyle().Foreground(colorBg).Background(colorGreen).Bold(true).Padding(0, 1),
		isoBadge: lipgloss.NewStyle().Foreground(colorDim).Italic(true),
		border: lipgloss.NewStyle().
			BorderLeft(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(colorAccent).
			PaddingLeft(1),
	}
}

func unselectedDelegateStyles() delegateStyles {
	return delegateStyles{
		name:     lipgloss.NewStyle().Foreground(colorFg),
		outBadge: lipgloss.NewStyle().Foreground(colorYellow).Bold(true),
		inBadge:  lipgloss.NewStyle().Foreground(colorGreen).Bold(true),
		isoBadge: lipgloss.NewStyle().Foreground(colorDim).Italic(true),
		border: lipgloss.NewStyle().
			BorderLeft(true).
			BorderStyle(lipgloss.HiddenBorder()).
			PaddingLeft(1),
	}
}

func renderBadges(svc scanner.Service, selected bool, ds delegateStyles, allSvc []scanner.Service, nameIdx *scanner.NameIndex) string {
	out := len(svc.Integrations)
	in := len(svc.DependedOnBy)

	var badges string
	if out > 0 {
		label := fmt.Sprintf("↑ %d", out)
		if selected {
			label = fmt.Sprintf("↑ %d out", out)
		}
		badges += "  " + ds.outBadge.Render(label)
	}
	if in > 0 {
		label := fmt.Sprintf("↓ %d", in)
		if selected {
			label = fmt.Sprintf("↓ %d in", in)
		}
		badges += "  " + ds.inBadge.Render(label)
	}
	if out == 0 && in == 0 {
		badges = "  " + ds.isoBadge.Render("○ isolated")
	}

	if staleCount := scanner.StaleCount(&svc, allSvc, nameIdx); staleCount > 0 {
		staleBadge := lipgloss.NewStyle().Foreground(colorYellow).Bold(true)
		if selected {
			staleBadge = lipgloss.NewStyle().Foreground(colorBg).Background(colorYellow).Bold(true).Padding(0, 1)
		}
		badges += "  " + staleBadge.Render(fmt.Sprintf("⚠ %d stale", staleCount))
	}

	return badges
}

func renderDepPreviewLine(svc scanner.Service, selected bool, arrowColor color.Color, arrow string, integrations []scanner.Integration, names []string) string {
	depColor := colorFg
	if !selected {
		depColor = colorDim
	}
	arrowStr := lipgloss.NewStyle().Foreground(arrowColor).Render(arrow)
	return arrowStr + " " + renderDepList(integrations, names, depColor, 70)
}

func padLines(lines string, lineCount, target int) string {
	for lineCount < target {
		lines += "\n"
		lineCount++
	}
	return lines
}

func (d serviceDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(serviceItem)
	if !ok {
		return
	}

	svc := item.service
	selected := index == m.Index()
	width := m.Width()

	ds := unselectedDelegateStyles()
	if selected {
		ds = selectedDelegateStyles()
	}

	allSvc := allServices(m)
	line1 := ds.name.Render(svc.Name) + renderBadges(svc, selected, ds, allSvc, d.nameIdx)

	lineCount := 1
	lines := line1

	if len(svc.Integrations) > 0 {
		lines += "\n" + renderDepPreviewLine(svc, selected, colorYellow, "→", svc.Integrations, nil)
		lineCount++
	}
	if len(svc.DependedOnBy) > 0 {
		lines += "\n" + renderDepPreviewLine(svc, selected, colorGreen, "←", nil, svc.DependedOnBy)
		lineCount++
	}

	lines = padLines(lines, lineCount, 3)
	fmt.Fprint(w, ds.border.Width(width-4).Render(lines))
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

func renderDetailHeader(svc *scanner.Service) string {
	var s strings.Builder
	title := svc.Name
	if svc.Version != "" {
		title += " · " + svc.Version
	}
	s.WriteString(detailTitleStyle.Render(fmt.Sprintf("  %s", title)) + "\n")
	s.WriteString(detailPathStyle.Render(svc.Path) + "\n")

	outCount := len(svc.Integrations)
	inCount := len(svc.DependedOnBy)
	statsLine := "  " +
		badgeOutStyle.Render(fmt.Sprintf(" %d outbound", outCount)) +
		"  " +
		badgeInStyle.Render(fmt.Sprintf(" %d inbound", inCount))
	s.WriteString(statsLine + "\n")
	return s.String()
}

func findNavIndex(integ scanner.Integration, navigable []scanner.Integration) int {
	for i, n := range navigable {
		if n.ClientID == integ.ClientID {
			return i
		}
	}
	return -1
}

func renderIntegrationLine(integ scanner.Integration, isInternal, isCursor bool, services []scanner.Service, nameIdx *scanner.NameIndex) string {
	var prefix, line string
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
		line += renderVersionAnnotation(integ, services, nameIdx)
	}
	return prefix + line + "\n"
}

func renderDetailOutbound(svc *scanner.Service, m ExploreModel) string {
	var s strings.Builder
	outCount := len(svc.Integrations)

	s.WriteString(
		sectionHeaderStyle.Render(
			arrowOutStyle.Render("")+"  Depends on "+
				sectionCountStyle.Render(fmt.Sprintf("(%d)", outCount)),
		) + "\n",
	)

	if outCount == 0 {
		s.WriteString(noneStyle.Render("  no outbound dependencies") + "\n")
		return s.String()
	}

	navigable := getNavigableIntegrations(svc.Integrations, m.nameIdx)
	for _, integ := range svc.Integrations {
		isInternal := m.nameIdx.IsInternal(integ.ClientID)
		navIdx := findNavIndex(integ, navigable)
		isCursor := navIdx >= 0 && navIdx == m.detailCursor
		s.WriteString(renderIntegrationLine(integ, isInternal, isCursor, m.services, m.nameIdx))
	}
	return s.String()
}

func renderDependentLine(dep string, isCursor bool, svc *scanner.Service, services []scanner.Service, nameIdx *scanner.NameIndex) string {
	var prefix, line string
	if isCursor {
		prefix = "  " + cursorStyle.Render("") + " "
		line = cursorStyle.Render(dep)
	} else {
		prefix = "  " + arrowInStyle.Render("") + " "
		line = internalStyle.Render(dep)
	}
	line += renderReverseStaleAnnotation(dep, svc, services, nameIdx)
	return prefix + line + "\n"
}

func renderDetailInbound(svc *scanner.Service, m ExploreModel) string {
	var s strings.Builder
	inCount := len(svc.DependedOnBy)

	s.WriteString(
		sectionHeaderStyle.Render(
			arrowInStyle.Render("")+"  Depended on by "+
				sectionCountStyle.Render(fmt.Sprintf("(%d)", inCount)),
		) + "\n",
	)

	if inCount == 0 {
		s.WriteString(noneStyle.Render("  no inbound dependencies") + "\n")
		return s.String()
	}

	navigable := getNavigableIntegrations(svc.Integrations, m.nameIdx)
	navOffset := len(navigable)
	for i, dep := range svc.DependedOnBy {
		isCursor := m.detailCursor == navOffset+i
		s.WriteString(renderDependentLine(dep, isCursor, svc, m.services, m.nameIdx))
	}
	return s.String()
}

func renderDetailFooter(m ExploreModel) string {
	if m.enteringHops {
		return renderHopInputFooter(m)
	}
	return renderNormalFooter(m)
}

func renderHopInputFooter(m ExploreModel) string {
	prompt := lipgloss.NewStyle().Bold(true).Foreground(colorAccent).Render("  Hops: ")
	val := lipgloss.NewStyle().Bold(true).Foreground(colorFg).Render(m.hopInput)
	cursor := lipgloss.NewStyle().Foreground(colorAccent).Render("█")
	return prompt + val + cursor + "  " + dimStyle.Render("enter to export · esc to cancel")
}

func renderNormalFooter(m ExploreModel) string {
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
	footer := curd.RenderHintBar(st, hints)
	if m.exportFlash != "" {
		flashStyle := lipgloss.NewStyle().Foreground(colorGreen).Bold(true).PaddingLeft(2)
		if strings.HasPrefix(m.exportFlash, "✗") {
			flashStyle = lipgloss.NewStyle().Foreground(colorRed).Bold(true).PaddingLeft(2)
		}
		footer = flashStyle.Render(m.exportFlash) + "\n" + footer
	}
	return footer
}

func renderDetail(m ExploreModel) string {
	svc := m.selectedService
	if svc == nil {
		return ""
	}

	contentWidth := max(m.width-8, 40)

	var content strings.Builder
	content.WriteString(renderDetailHeader(svc))
	content.WriteString(renderDetailOutbound(svc, m))
	content.WriteString(renderDetailInbound(svc, m))

	box := detailBoxStyle.Width(contentWidth).Render(content.String())
	footer := renderDetailFooter(m)

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
		if m.enteringHops {
			return handleHopInput(m, msg)
		}
		return handleDetailNavKey(m, msg)
	case GraphExportedMsg:
		return handleGraphExported(m, msg), nil
	}
	return m, nil
}

func handleHopInput(m ExploreModel, msg tea.KeyPressMsg) (ExploreModel, tea.Cmd) {
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

func handleDetailNavKey(m ExploreModel, msg tea.KeyPressMsg) (ExploreModel, tea.Cmd) {
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
		return handleDetailEnter(m), nil
	}
	return m, nil
}

func handleDetailEnter(m ExploreModel) ExploreModel {
	targetName := resolveDetailTarget(m)
	if targetName == "" {
		return m
	}
	for i := range m.services {
		if m.services[i].Name == targetName {
			m.pushNav()
			m.selectedService = &m.services[i]
			m.detailCursor = 0
			return m
		}
	}
	return m
}

func handleGraphExported(m ExploreModel, msg GraphExportedMsg) ExploreModel {
	if msg.Err != nil {
		m.exportFlash = "✗ " + msg.Err.Error()
	} else {
		m.exportFlash = "✓ opened graph"
	}
	return m
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
