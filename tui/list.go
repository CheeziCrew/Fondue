package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cheezi/service-map/scanner"
)

// serviceItem implements list.Item for the bubbles list component.
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

// ── Custom delegate ─────────────────────────────────────────────────

type serviceDelegate struct{}

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

	// ── Line 1: service name + badges ───────────────────────────
	var nameStyle, outBadge, inBadge, isoBadge lipgloss.Style
	var border lipgloss.Style

	if selected {
		nameStyle = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
		outBadge = lipgloss.NewStyle().Foreground(colorBg).Background(colorMagenta).Bold(true).Padding(0, 1)
		inBadge = lipgloss.NewStyle().Foreground(colorBg).Background(colorGreen).Bold(true).Padding(0, 1)
		isoBadge = lipgloss.NewStyle().Foreground(colorDim).Italic(true)
		border = lipgloss.NewStyle().
			BorderLeft(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(colorAccent).
			PaddingLeft(1)
	} else {
		nameStyle = lipgloss.NewStyle().Foreground(colorFg)
		outBadge = lipgloss.NewStyle().Foreground(colorMagenta).Bold(true)
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

	// Stale badge — need access to all services for comparison
	staleCount := scanner.StaleCount(&svc, allServices(m))
	if staleCount > 0 {
		staleBadge := lipgloss.NewStyle().Foreground(colorYellow).Bold(true)
		if selected {
			staleBadge = lipgloss.NewStyle().Foreground(colorBg).Background(colorYellow).Bold(true).Padding(0, 1)
		}
		badges += "  " + staleBadge.Render(fmt.Sprintf("⚠ %d stale", staleCount))
	}

	line1 := name + badges

	// ── Line 2: outbound deps ───────────────────────────────────
	const maxPreview = 70
	var line2 string
	if out > 0 {
		arrow := lipgloss.NewStyle().Foreground(colorMagenta).Render("→")
		depColor := colorFg
		if !selected {
			depColor = colorDim
		}
		line2 = arrow + " " + renderDepList(svc.Integrations, nil, depColor, maxPreview)
	}

	// ── Line 3: inbound deps ────────────────────────────────────
	var line3 string
	if in > 0 {
		arrow := lipgloss.NewStyle().Foreground(colorGreen).Render("←")
		depColor := colorFg
		if !selected {
			depColor = colorDim
		}
		line3 = arrow + " " + renderDepList(nil, svc.DependedOnBy, depColor, maxPreview)
	}

	// ── Compose ─────────────────────────────────────────────────
	lines := line1
	if line2 != "" {
		lines += "\n" + line2
	}
	if line3 != "" {
		lines += "\n" + line3
	}

	// Pad missing lines to maintain consistent height
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

// renderDepList styles each dep name individually so separators don't break colors.
func renderDepList(integrations []scanner.Integration, names []string, color lipgloss.Color, maxWidth int) string {
	depStyle := lipgloss.NewStyle().Foreground(color)
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

// allServices extracts all services from the list model's items.
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

// substringFilter matches items whose filter value contains the search term
// as a case-insensitive substring. No fuzzy matching.
func substringFilter(term string, targets []string) []list.Rank {
	term = strings.ToLower(term)
	var ranks []list.Rank
	for i, t := range targets {
		lower := strings.ToLower(t)
		idx := strings.Index(lower, term)
		if idx >= 0 {
			// Build matched indices for highlighting
			matchedIndexes := make([]int, len(term))
			for j := range term {
				matchedIndexes[j] = idx + j
			}
			ranks = append(ranks, list.Rank{
				Index:          i,
				MatchedIndexes: matchedIndexes,
			})
		}
	}
	return ranks
}

func newServiceList(services []scanner.Service, width, height int) list.Model {
	items := make([]list.Item, len(services))
	for i, s := range services {
		items[i] = serviceItem{service: s}
	}

	l := list.New(items, serviceDelegate{}, width, height)
	l.Filter = substringFilter
	l.Title = "  Service Map"
	l.Styles.Title = listTitleStyle
	l.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	l.Styles.FilterCursor = lipgloss.NewStyle().Foreground(colorAccent)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.StatusBar = lipgloss.NewStyle().Foreground(colorDim).PaddingLeft(2)

	return l
}

func handleListUpdate(m Model, msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't handle enter while filtering
		if m.list.FilterState() == list.Filtering {
			break
		}
		switch msg.String() {
		case "enter":
			if item, ok := m.list.SelectedItem().(serviceItem); ok {
				m.selectedService = &item.service
				m.view = detailView
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}
