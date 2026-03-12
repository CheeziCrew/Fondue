package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/CheeziCrew/curd"
	"github.com/CheeziCrew/fondue/graph"
	"github.com/CheeziCrew/fondue/scanner"
	"github.com/CheeziCrew/fondue/tui/screens"
)

type screen int

const (
	screenMenu screen = iota
	screenExplore
	screenStats
	screenStale
)

// Model is the root Bubble Tea model that routes to sub-screens.
type Model struct {
	current  screen
	services []scanner.Service
	nameIdx  *scanner.NameIndex
	menu     screens.MenuModel
	explore  screens.ExploreModel
	stats    screens.StatsModel
	stale    screens.StaleModel
	width    int
	height   int
}

func NewModel(services []scanner.Service, idx *scanner.NameIndex) Model {
	return Model{
		current:  screenMenu,
		services: services,
		nameIdx:  idx,
		menu:     screens.NewMenu(),
		width:    80,
		height:   24,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Forward to all active screens.
		m.menu, _ = m.menu.Update(msg)
		if m.current == screenExplore {
			m.explore, _ = m.explore.Update(msg)
		}
		if m.current == screenStats {
			m.stats, _ = m.stats.Update(msg)
		}
		if m.current == screenStale {
			m.stale, _ = m.stale.Update(msg)
		}
		return m, nil

	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if msg.String() == "q" && m.current == screenMenu {
			return m, tea.Quit
		}

	case curd.MenuSelectionMsg:
		return m, m.handleMenuSelection(msg)

	case screens.BackToMenuMsg:
		m.current = screenMenu
		m.menu = screens.NewMenu()
		return m, func() tea.Msg {
			return tea.WindowSizeMsg{Width: m.width, Height: m.height}
		}

	case screens.ExportGraphMsg:
		return m, m.exportSubgraph(msg)

	case screens.GraphExportedMsg:
		// Forward to explore screen so it can show the flash.
		m.explore, _ = m.explore.Update(msg)
		return m, nil
	}

	// Delegate to current screen.
	var cmd tea.Cmd
	switch m.current {
	case screenMenu:
		m.menu, cmd = m.menu.Update(msg)
	case screenExplore:
		m.explore, cmd = m.explore.Update(msg)
	case screenStats:
		m.stats, cmd = m.stats.Update(msg)
	case screenStale:
		m.stale, cmd = m.stale.Update(msg)
	}
	return m, cmd
}

func (m *Model) exportSubgraph(msg screens.ExportGraphMsg) tea.Cmd {
	services := m.services
	idx := m.nameIdx
	return func() tea.Msg {
		_, err := graph.ExportPNG(msg.Service, services, idx, msg.Hops)
		return screens.GraphExportedMsg{Err: err}
	}
}

func (m *Model) handleMenuSelection(msg curd.MenuSelectionMsg) tea.Cmd {
	sizeCmd := func() tea.Msg {
		return tea.WindowSizeMsg{Width: m.width, Height: m.height}
	}

	switch msg.Command {
	case "explore":
		m.current = screenExplore
		m.explore = screens.NewExplore(m.services, m.nameIdx, m.width, m.height)
		return sizeCmd
	case "stats":
		m.current = screenStats
		m.stats = screens.NewStats(m.services, m.nameIdx, m.width, m.height)
		return sizeCmd
	case "stale":
		m.current = screenStale
		m.stale = screens.NewStale(m.services, m.nameIdx, m.width, m.height)
		return sizeCmd
	}
	return nil
}

func (m Model) View() tea.View {
	var content string
	switch m.current {
	case screenMenu:
		content = m.menu.View()
	case screenExplore:
		content = m.explore.View()
	case screenStats:
		content = m.stats.View()
	case screenStale:
		content = m.stale.View()
	}
	v := tea.NewView(lipgloss.NewStyle().Padding(1, 2, 0, 2).Render(content))
	v.AltScreen = true
	v.WindowTitle = "fondue"
	return v
}
