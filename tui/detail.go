package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cheezi/service-map/scanner"
)

func renderDetail(m Model) string {
	svc := m.selectedService
	if svc == nil {
		return ""
	}

	contentWidth := m.width - 8 // account for box padding + margins
	if contentWidth < 40 {
		contentWidth = 40
	}

	var content strings.Builder

	// ── Header ──────────────────────────────────────────────────────
	title := svc.Name
	if svc.Version != "" {
		title += " · " + svc.Version
	}
	titleLine := detailTitleStyle.Render(fmt.Sprintf("  %s", title))
	content.WriteString(titleLine + "\n")
	content.WriteString(detailPathStyle.Render(svc.Path) + "\n")

	// Version line
	if svc.Version != "" {
		content.WriteString(versionDimStyle.Render("  v"+svc.Version) + "\n")
	}

	// Stats bar
	outCount := len(svc.Integrations)
	inCount := len(svc.DependedOnBy)
	statsLine := "  " +
		badgeOutStyle.Render(fmt.Sprintf(" %d outbound", outCount)) +
		"  " +
		badgeInStyle.Render(fmt.Sprintf(" %d inbound", inCount))
	content.WriteString(statsLine + "\n")

	// ── Depends on ──────────────────────────────────────────────────
	content.WriteString(
		sectionHeaderStyle.Render(
			arrowOutStyle.Render("") + "  Depends on " +
				sectionCountStyle.Render(fmt.Sprintf("(%d)", outCount)),
		) + "\n",
	)

	if outCount == 0 {
		content.WriteString(noneStyle.Render("  no outbound dependencies") + "\n")
	} else {
		navigable := getNavigableIntegrations(svc.Integrations, m.services)
		for _, integ := range svc.Integrations {
			isInternal := scanner.IsInternal(integ.ClientID, m.services)

			// Check if this item is the cursor target
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

			// Version annotation for internal integrations
			if isInternal {
				line += renderVersionAnnotation(integ, m.services)
			}

			content.WriteString(prefix + line + "\n")
		}
	}

	// ── Depended on by ──────────────────────────────────────────────
	content.WriteString(
		sectionHeaderStyle.Render(
			arrowInStyle.Render("") + "  Depended on by " +
				sectionCountStyle.Render(fmt.Sprintf("(%d)", inCount)),
		) + "\n",
	)

	if inCount == 0 {
		content.WriteString(noneStyle.Render("  no inbound dependencies") + "\n")
	} else {
		navigable := getNavigableIntegrations(svc.Integrations, m.services)
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

			content.WriteString(prefix + line + "\n")
		}
	}

	// ── Wrap in box ─────────────────────────────────────────────────
	box := detailBoxStyle.Width(contentWidth).Render(content.String())

	// ── Help bar ────────────────────────────────────────────────────
	help := renderHintBar()

	return lipgloss.JoinVertical(lipgloss.Left, box, help)
}

func renderHintBar() string {
	hints := []struct{ key, desc string }{
		{"esc", "back"},
		{"enter", "navigate"},
		{"j/k", "move"},
		{"q", "quit"},
	}

	var parts []string
	for _, h := range hints {
		parts = append(parts,
			hintKeyStyle.Render(h.key)+" "+hintDescStyle.Render(h.desc),
		)
	}

	return hintBarStyle.Render(
		strings.Join(parts, hintSepStyle.Render("  ·  ")),
	)
}

func renderVersionAnnotation(integ scanner.Integration, services []scanner.Service) string {
	if integ.SpecVersion == "" {
		return ""
	}

	targetName := scanner.ResolveTargetName(integ.ClientID, services)
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

func getNavigableIntegrations(integrations []scanner.Integration, services []scanner.Service) []scanner.Integration {
	var nav []scanner.Integration
	for _, integ := range integrations {
		if scanner.IsInternal(integ.ClientID, services) {
			nav = append(nav, integ)
		}
	}
	return nav
}

func getNavigableCount(svc *scanner.Service, services []scanner.Service) int {
	return len(getNavigableIntegrations(svc.Integrations, services)) + len(svc.DependedOnBy)
}

func handleDetailUpdate(m Model, msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		maxItems := getNavigableCount(m.selectedService, m.services)

		switch msg.String() {
		case "esc", "backspace":
			m.view = listView
			m.detailCursor = 0
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

		case "enter":
			targetName := resolveDetailTarget(m)
			if targetName != "" {
				for i := range m.services {
					if m.services[i].Name == targetName {
						m.selectedService = &m.services[i]
						m.detailCursor = 0
						return m, nil
					}
				}
			}
			return m, nil

		case "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func resolveDetailTarget(m Model) string {
	svc := m.selectedService
	navigable := getNavigableIntegrations(svc.Integrations, m.services)
	navOffset := len(navigable)

	if m.detailCursor < navOffset {
		clientID := navigable[m.detailCursor].ClientID
		nameMap := make(map[string]string)
		for _, s := range m.services {
			nameMap[s.Name] = s.Name
			nameMap[strings.ReplaceAll(s.Name, "-", "")] = s.Name
		}
		lower := strings.ToLower(clientID)
		if name, ok := nameMap[lower]; ok {
			return name
		}
		stripped := strings.ReplaceAll(lower, "-", "")
		if name, ok := nameMap[stripped]; ok {
			return name
		}
	} else {
		idx := m.detailCursor - navOffset
		if idx < len(svc.DependedOnBy) {
			return svc.DependedOnBy[idx]
		}
	}
	return ""
}
