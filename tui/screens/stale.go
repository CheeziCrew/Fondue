package screens

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/CheeziCrew/curd"
	"github.com/CheeziCrew/fondue/scanner"
)

// StaleModel shows all stale integration specs across services.
type StaleModel struct {
	viewport  viewport.Model
	viewReady bool
	content   string
	width     int
	height    int
}

func NewStale(services []scanner.Service, idx *scanner.NameIndex, width, height int) StaleModel {
	content := renderStaleOverview(services, idx)
	m := StaleModel{content: content, width: width, height: height}
	if width > 0 && height > 0 {
		m.viewport = viewport.New(viewport.WithWidth(width-8), viewport.WithHeight(height-6))
		m.viewport.SetContent(content)
		m.viewReady = true
	}
	return m
}

func (m StaleModel) Init() tea.Cmd { return nil }

func (m StaleModel) Update(msg tea.Msg) (StaleModel, tea.Cmd) {
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

func (m StaleModel) View() string {
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

type staleEntry struct {
	repo        string
	specFile    string
	specVersion string
	targetVer   string
}

func collectStaleEntries(services []scanner.Service, idx *scanner.NameIndex) []staleEntry {
	var entries []staleEntry
	for _, svc := range services {
		entries = append(entries, collectServiceStaleEntries(svc, services, idx)...)
	}
	return entries
}

func collectServiceStaleEntries(svc scanner.Service, services []scanner.Service, idx *scanner.NameIndex) []staleEntry {
	var entries []staleEntry
	for _, integ := range svc.Integrations {
		if integ.SpecVersion == "" {
			continue
		}
		targetName := idx.Resolve(integ.ClientID)
		if targetName == "" {
			continue
		}
		target := scanner.FindService(targetName, services)
		if target == nil || target.Version == "" {
			continue
		}
		if integ.SpecVersion != target.Version {
			entries = append(entries, staleEntry{
				repo:        svc.Name,
				specFile:    integ.ClientID,
				specVersion: integ.SpecVersion,
				targetVer:   target.Version,
			})
		}
	}
	return entries
}

func groupByRepo(entries []staleEntry) (map[string][]staleEntry, []string) {
	byRepo := make(map[string][]staleEntry)
	var repoOrder []string
	for _, e := range entries {
		if _, seen := byRepo[e.repo]; !seen {
			repoOrder = append(repoOrder, e.repo)
		}
		byRepo[e.repo] = append(byRepo[e.repo], e)
	}
	return byRepo, repoOrder
}

func renderStaleRepoGroup(repo string, entries []staleEntry) string {
	var s strings.Builder
	unresolvedStyle := lipgloss.NewStyle().Foreground(colorRed).Bold(true)

	s.WriteString("  " + internalStyle.Render(repo) + "\n")
	for _, e := range entries {
		specVer := staleStyle.Render(e.specVersion)
		if isUnresolvedPlaceholder(e.specVersion) {
			specVer = unresolvedStyle.Render("unresolved")
		}
		s.WriteString(fmt.Sprintf("    %s  %s → %s\n",
			dimStyle.Render(e.specFile),
			specVer,
			lipgloss.NewStyle().Foreground(colorGreen).Render(e.targetVer),
		))
	}
	s.WriteString("\n")
	return s.String()
}

func renderStaleOverview(services []scanner.Service, idx *scanner.NameIndex) string {
	entries := collectStaleEntries(services, idx)

	var s strings.Builder
	s.WriteString(detailTitleStyle.Render("  ⚠️  Stale Integration Specs") + "\n\n")

	if len(entries) == 0 {
		s.WriteString("  " + lipgloss.NewStyle().Foreground(colorGreen).Bold(true).Render("✓") + " All integration specs are up to date.\n")
		return s.String()
	}

	countStyle := lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	s.WriteString("  " + countStyle.Render(fmt.Sprintf("%d", len(entries))) + " stale spec(s) found\n\n")

	byRepo, repoOrder := groupByRepo(entries)
	for _, repo := range repoOrder {
		s.WriteString(renderStaleRepoGroup(repo, byRepo[repo]))
	}

	return s.String()
}

func isUnresolvedPlaceholder(v string) bool {
	return strings.Contains(v, "@") || strings.Contains(v, "${")
}
