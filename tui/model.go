package tui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/CheeziCrew/fondue/scanner"
)

type viewState int

const (
	listView viewState = iota
	detailView
)

// navEntry stores state for back-navigation.
type navEntry struct {
	service *scanner.Service
	cursor  int
}

type Model struct {
	services        []scanner.Service
	nameIdx         *scanner.NameIndex
	list            list.Model
	view            viewState
	selectedService *scanner.Service
	detailCursor    int
	navStack        []navEntry // breadcrumb history
	width           int
	height          int
}

func NewModel(services []scanner.Service, idx *scanner.NameIndex) Model {
	m := Model{
		services: services,
		nameIdx:  idx,
		view:     listView,
		width:    80,
		height:   24,
	}
	m.list = newServiceList(services, idx, m.width, m.height)
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

// pushNav saves current detail state to the nav stack.
func (m *Model) pushNav() {
	m.navStack = append(m.navStack, navEntry{
		service: m.selectedService,
		cursor:  m.detailCursor,
	})
}

// popNav restores the previous detail state. Returns false if stack is empty.
func (m *Model) popNav() bool {
	if len(m.navStack) == 0 {
		return false
	}
	last := m.navStack[len(m.navStack)-1]
	m.navStack = m.navStack[:len(m.navStack)-1]
	m.selectedService = last.service
	m.detailCursor = last.cursor
	return true
}
