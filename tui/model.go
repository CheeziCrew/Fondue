package tui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cheezi/service-map/scanner"
)

type viewState int

const (
	listView viewState = iota
	detailView
)

type Model struct {
	services        []scanner.Service
	list            list.Model
	view            viewState
	selectedService *scanner.Service
	detailCursor    int
	width           int
	height          int
}

func NewModel(services []scanner.Service) Model {
	m := Model{
		services: services,
		view:     listView,
		width:    80,
		height:   24,
	}
	m.list = newServiceList(services, m.width, m.height)
	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	switch m.view {
	case listView:
		return handleListUpdate(m, msg)
	case detailView:
		return handleDetailUpdate(m, msg)
	}

	return m, nil
}

func (m Model) View() string {
	switch m.view {
	case detailView:
		return renderDetail(m)
	default:
		return m.list.View()
	}
}
