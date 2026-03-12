package screens

import (
	"fmt"
	"image/color"
	"sort"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/CheeziCrew/curd"
	"github.com/CheeziCrew/fondue/graph"
	"github.com/CheeziCrew/fondue/scanner"
)

// StatsModel shows an overview of services, connectivity, and stale specs.
type StatsModel struct {
	viewport  viewport.Model
	viewReady bool
	content   string
	width     int
	height    int
}

func NewStats(services []scanner.Service, idx *scanner.NameIndex, width, height int) StatsModel {
	content := renderStats(services, idx, width)
	m := StatsModel{content: content, width: width, height: height}
	if width > 0 && height > 0 {
		m.viewport = viewport.New(viewport.WithWidth(width-8), viewport.WithHeight(height-6))
		m.viewport.SetContent(content)
		m.viewReady = true
	}
	return m
}

func (m StatsModel) Init() tea.Cmd { return nil }

func (m StatsModel) Update(msg tea.Msg) (StatsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.viewReady {
			m.viewport = viewport.New(viewport.WithWidth(msg.Width-8), viewport.WithHeight(msg.Height-6))
			m.viewport.SetContent(m.content)
			m.viewReady = true
		} else {
			m.viewport.SetWidth(msg.Width - 8)
			m.viewport.SetHeight(msg.Height - 6)
		}
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc", "q":
			return m, func() tea.Msg { return BackToMenuMsg{} }
		}
	}

	if m.viewReady {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m StatsModel) View() string {
	if !m.viewReady {
		return ""
	}

	vp := m.viewport.View()

	hints := []curd.Hint{{Key: "esc/q", Desc: "menu"}}
	if m.viewport.TotalLineCount() > m.viewport.VisibleLineCount() {
		hints = append([]curd.Hint{{Key: "↑/↓", Desc: "scroll"}}, hints...)
	}
	help := curd.RenderHintBar(st, hints)

	return lipgloss.JoinVertical(lipgloss.Left, vp, help)
}

// ── Data collection ────────────────────────────────────────────────

type ranked struct {
	name  string
	count int
}

type statsData struct {
	totalServices int
	totalOut      int
	totalIn       int
	totalStale    int
	isolated      []string
	topOutbound   []ranked
	topInbound    []ranked
	staleHotspots []ranked
	// Distribution buckets: 0, 1-2, 3-5, 6-10, 11+
	distribution [5]int
}

func collectStats(services []scanner.Service, idx *scanner.NameIndex) statsData {
	d := statsData{totalServices: len(services)}

	for _, svc := range services {
		out := len(svc.Integrations)
		in := len(svc.DependedOnBy)
		d.totalOut += out
		d.totalIn += in

		stale := scanner.StaleCount(&svc, services, idx)
		d.totalStale += stale

		if out == 0 && in == 0 {
			d.isolated = append(d.isolated, svc.Name)
		}

		d.topOutbound = append(d.topOutbound, ranked{svc.Name, out})
		d.topInbound = append(d.topInbound, ranked{svc.Name, in})
		if stale > 0 {
			d.staleHotspots = append(d.staleHotspots, ranked{svc.Name, stale})
		}

		// Distribution
		total := out + in
		switch {
		case total == 0:
			d.distribution[0]++
		case total <= 2:
			d.distribution[1]++
		case total <= 5:
			d.distribution[2]++
		case total <= 10:
			d.distribution[3]++
		default:
			d.distribution[4]++
		}
	}

	sortRanked := func(r []ranked) {
		sort.Slice(r, func(i, j int) bool { return r[i].count > r[j].count })
	}
	sortRanked(d.topOutbound)
	sortRanked(d.topInbound)
	sortRanked(d.staleHotspots)

	return d
}

// ── Rendering ──────────────────────────────────────────────────────

func renderStats(services []scanner.Service, idx *scanner.NameIndex, width int) string {
	d := collectStats(services, idx)

	contentWidth := width - 12
	if contentWidth < 50 {
		contentWidth = 50
	}

	var s strings.Builder

	// Title
	s.WriteString(detailTitleStyle.Render("  📊 Service Overview") + "\n\n")

	// ── Summary cards ──
	s.WriteString(renderSummaryCards(d) + "\n")

	// ── Distribution ──
	s.WriteString(renderDistribution(d) + "\n")

	// ── Top Outbound ──
	s.WriteString(sectionHeaderStyle.Render("  ↑ Top Outbound") + "\n")
	s.WriteString(renderBarChart(d.topOutbound, contentWidth, colorYellow) + "\n")

	// ── Top Inbound ──
	s.WriteString(sectionHeaderStyle.Render("  ↓ Top Inbound") + "\n")
	s.WriteString(renderBarChart(d.topInbound, contentWidth, colorGreen) + "\n")

	// ── Stale Hotspots ──
	if len(d.staleHotspots) > 0 {
		s.WriteString(sectionHeaderStyle.Render("  ⚠ Stale Hotspots") + "\n")
		s.WriteString(renderStaleChart(d.staleHotspots, contentWidth) + "\n")
	}

	// ── Isolated ──
	if len(d.isolated) > 0 {
		s.WriteString(sectionHeaderStyle.Render(fmt.Sprintf("  ○ Isolated (%d)", len(d.isolated))) + "\n")
		s.WriteString(renderIsolated(d.isolated, contentWidth))
	}

	return s.String()
}

func renderSummaryCards(d statsData) string {
	card := func(label string, value string, borderColor color.Color) string {
		valueStyle := lipgloss.NewStyle().Bold(true).Foreground(borderColor).Align(lipgloss.Center)
		labelStyle := lipgloss.NewStyle().Foreground(colorDim).Align(lipgloss.Center)
		box := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(0, 2).
			Width(16).
			Align(lipgloss.Center)
		return box.Render(valueStyle.Render(value) + "\n" + labelStyle.Render(label))
	}

	svcCard := card("services", fmt.Sprintf("%d", d.totalServices), colorAccent)
	outCard := card("outbound", fmt.Sprintf("%d", d.totalOut), colorYellow)
	inCard := card("inbound", fmt.Sprintf("%d", d.totalIn), colorGreen)

	cards := lipgloss.JoinHorizontal(lipgloss.Top, "  ", svcCard, "  ", outCard, "  ", inCard)

	if d.totalStale > 0 {
		staleCard := card("stale", fmt.Sprintf("⚠ %d", d.totalStale), colorRed)
		cards = lipgloss.JoinHorizontal(lipgloss.Top, cards, "  ", staleCard)
	}

	return cards
}

func renderDistribution(d statsData) string {
	labels := []string{"0", "1-2", "3-5", "6-10", "11+"}

	maxBucket := 0
	for _, v := range d.distribution {
		if v > maxBucket {
			maxBucket = v
		}
	}
	if maxBucket == 0 {
		return ""
	}

	// Vertical bar chart: render rows top-down, then labels, then counts
	barHeight := 6
	colWidth := 8

	var s strings.Builder
	s.WriteString(sectionHeaderStyle.Render("  ◆ Connectivity Distribution") + "\n")

	// Render bars top-down
	for row := barHeight; row >= 1; row-- {
		s.WriteString("  ")
		threshold := float64(row) / float64(barHeight)
		for i, count := range d.distribution {
			t := float64(count) / float64(maxBucket)
			hc := graph.HeatColor(float64(i) / float64(len(d.distribution)-1))
			cell := lipgloss.NewStyle().Width(colWidth).Align(lipgloss.Center)

			if t >= threshold {
				block := lipgloss.NewStyle().Foreground(lipgloss.Color(hc)).Render("██")
				s.WriteString(cell.Render(block))
			} else if t > 0 && float64(row) == 1 {
				// Ensure non-zero always shows at least bottom row
				block := lipgloss.NewStyle().Foreground(lipgloss.Color(hc)).Render("▄▄")
				s.WriteString(cell.Render(block))
			} else {
				s.WriteString(cell.Render(" "))
			}
		}
		s.WriteString("\n")
	}

	// Count row
	s.WriteString("  ")
	for _, count := range d.distribution {
		cell := lipgloss.NewStyle().Width(colWidth).Align(lipgloss.Center).Foreground(colorFg).Bold(true)
		s.WriteString(cell.Render(fmt.Sprintf("%d", count)))
	}
	s.WriteString("\n")

	// Label row
	s.WriteString("  ")
	for _, label := range labels {
		cell := lipgloss.NewStyle().Width(colWidth).Align(lipgloss.Center).Foreground(colorDim)
		s.WriteString(cell.Render(label))
	}
	s.WriteString("\n")
	s.WriteString("  " + dimStyle.Render("connections per service") + "\n")

	return s.String()
}

func renderBarChart(items []ranked, maxWidth int, baseColor color.Color) string {
	top := 5
	if len(items) < top {
		top = len(items)
	}
	// Filter out zero-count items
	filtered := make([]ranked, 0, top)
	for i := 0; i < top && i < len(items); i++ {
		if items[i].count > 0 {
			filtered = append(filtered, items[i])
		}
	}
	if len(filtered) == 0 {
		return "  " + dimStyle.Render("none") + "\n"
	}

	maxCount := filtered[0].count
	nameWidth := 0
	for _, r := range filtered {
		if len(r.name) > nameWidth {
			nameWidth = len(r.name)
		}
	}
	if nameWidth > 25 {
		nameWidth = 25
	}

	barMaxWidth := maxWidth - nameWidth - 12
	if barMaxWidth < 10 {
		barMaxWidth = 10
	}

	var s strings.Builder
	for i, r := range filtered {
		t := float64(r.count) / float64(maxCount)
		barLen := int(t * float64(barMaxWidth))
		if barLen < 1 && r.count > 0 {
			barLen = 1
		}

		// Heat color based on rank position
		hc := graph.HeatColor(float64(i) / float64(max(len(filtered)-1, 1)))
		barColor := lipgloss.Color(hc)
		bar := lipgloss.NewStyle().Foreground(barColor).Render(strings.Repeat("█", barLen))
		name := lipgloss.NewStyle().Foreground(baseColor).Width(nameWidth).Render(r.name)
		count := lipgloss.NewStyle().Foreground(colorFg).Bold(true).Render(fmt.Sprintf(" %d", r.count))

		s.WriteString("  " + name + " " + bar + count + "\n")
	}
	return s.String()
}

func renderStaleChart(items []ranked, maxWidth int) string {
	top := 5
	if len(items) < top {
		top = len(items)
	}

	maxCount := items[0].count
	nameWidth := 0
	for i := 0; i < top; i++ {
		if len(items[i].name) > nameWidth {
			nameWidth = len(items[i].name)
		}
	}
	if nameWidth > 25 {
		nameWidth = 25
	}

	barMaxWidth := maxWidth - nameWidth - 12
	if barMaxWidth < 10 {
		barMaxWidth = 10
	}

	var s strings.Builder
	for i := 0; i < top; i++ {
		r := items[i]
		t := float64(r.count) / float64(maxCount)
		barLen := int(t * float64(barMaxWidth))
		if barLen < 1 {
			barLen = 1
		}

		// Warm colors only for stale (0.5 → 1.0 range = yellow to red)
		ht := 0.5 + (float64(i)/float64(max(top-1, 1)))*0.5
		// Invert: top item (most stale) should be hottest
		ht = 1.0 - (float64(i)/float64(max(top-1, 1)))*0.5
		barColor := lipgloss.Color(graph.HeatColor(ht))
		bar := lipgloss.NewStyle().Foreground(barColor).Render(strings.Repeat("█", barLen))
		name := lipgloss.NewStyle().Foreground(colorYellow).Width(nameWidth).Render(r.name)
		count := staleStyle.Render(fmt.Sprintf(" ⚠ %d", r.count))

		s.WriteString("  " + name + " " + bar + count + "\n")
	}
	return s.String()
}

func renderIsolated(names []string, maxWidth int) string {
	if len(names) <= 6 {
		// Compact: comma-separated
		styled := make([]string, len(names))
		for i, n := range names {
			styled[i] = dimStyle.Render(n)
		}
		return "  " + dimStyle.Render("○ ") + strings.Join(styled, dimStyle.Render(" · ")) + "\n"
	}

	// Multi-line with columns
	var s strings.Builder
	for _, n := range names {
		s.WriteString("  " + dimStyle.Render("○ "+n) + "\n")
	}
	return s.String()
}
