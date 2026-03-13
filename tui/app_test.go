package tui

import (
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/CheeziCrew/curd"
	"github.com/CheeziCrew/fondue/scanner"
	"github.com/CheeziCrew/fondue/tui/screens"
)

func testServices() []scanner.Service {
	return []scanner.Service{
		{
			Name:    "auth-service",
			Path:    "/tmp/auth",
			Version: "1.0.0",
			Integrations: []scanner.Integration{
				{ClientID: "user-service", SpecVersion: "2.0.0"},
			},
		},
		{
			Name:         "user-service",
			Path:         "/tmp/user",
			Version:      "2.0.0",
			DependedOnBy: []string{"auth-service"},
		},
		{
			Name: "isolated",
			Path: "/tmp/isolated",
		},
	}
}

func testIdx() *scanner.NameIndex {
	return scanner.NewNameIndexFromServices(testServices())
}

func TestNewModel(t *testing.T) {
	m := NewModel(testServices(), testIdx())

	if m.current != screenMenu {
		t.Errorf("initial screen = %d, want screenMenu (%d)", m.current, screenMenu)
	}
	if m.width != 80 || m.height != 24 {
		t.Errorf("dimensions = %dx%d, want 80x24", m.width, m.height)
	}
	if len(m.services) != 3 {
		t.Errorf("services count = %d, want 3", len(m.services))
	}
}

func TestModelInit(t *testing.T) {
	m := NewModel(testServices(), testIdx())
	cmd := m.Init()
	if cmd != nil {
		t.Error("expected Init() to return nil")
	}
}

func TestModelViewMenu(t *testing.T) {
	m := NewModel(testServices(), testIdx())
	v := m.View()
	if v.WindowTitle != "fondue" {
		t.Errorf("WindowTitle = %q, want %q", v.WindowTitle, "fondue")
	}
	if !v.AltScreen {
		t.Error("expected AltScreen = true")
	}
}

func TestModelUpdateWindowSize(t *testing.T) {
	m := NewModel(testServices(), testIdx())
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}

	result, cmd := m.Update(msg)
	m2 := result.(Model)

	if m2.width != 120 || m2.height != 40 {
		t.Errorf("dimensions = %dx%d, want 120x40", m2.width, m2.height)
	}
	if cmd != nil {
		t.Error("expected nil cmd from window size update")
	}
}

func TestModelUpdateWindowSizeOnExplore(t *testing.T) {
	m := NewModel(testServices(), testIdx())
	m.current = screenExplore
	m.explore = screens.NewExplore(m.services, m.nameIdx, 80, 24)

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	result, _ := m.Update(msg)
	m2 := result.(Model)

	if m2.width != 120 {
		t.Errorf("width = %d, want 120", m2.width)
	}
}

func TestModelUpdateWindowSizeOnStats(t *testing.T) {
	m := NewModel(testServices(), testIdx())
	m.current = screenStats
	m.stats = screens.NewStats(m.services, m.nameIdx, 80, 24)

	msg := tea.WindowSizeMsg{Width: 100, Height: 30}
	result, _ := m.Update(msg)
	m2 := result.(Model)

	if m2.width != 100 {
		t.Errorf("width = %d, want 100", m2.width)
	}
}

func TestModelUpdateWindowSizeOnStale(t *testing.T) {
	m := NewModel(testServices(), testIdx())
	m.current = screenStale
	m.stale = screens.NewStale(m.services, m.nameIdx, 80, 24)

	msg := tea.WindowSizeMsg{Width: 100, Height: 30}
	result, _ := m.Update(msg)
	m2 := result.(Model)

	if m2.width != 100 {
		t.Errorf("width = %d, want 100", m2.width)
	}
}

func TestModelUpdateCtrlCQuits(t *testing.T) {
	m := NewModel(testServices(), testIdx())
	msg := tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}

	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Fatal("expected quit command")
	}
	result := cmd()
	if _, ok := result.(tea.QuitMsg); !ok {
		t.Errorf("expected QuitMsg, got %T", result)
	}
}

func TestModelUpdateQuitOnMenu(t *testing.T) {
	m := NewModel(testServices(), testIdx())
	m.current = screenMenu
	msg := tea.KeyPressMsg{Code: 'q', Text: "q"}

	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Fatal("expected quit command on q from menu")
	}
}

func TestModelMenuSelectionExplore(t *testing.T) {
	m := NewModel(testServices(), testIdx())
	msg := curd.MenuSelectionMsg{Command: "explore"}

	result, cmd := m.Update(msg)
	m2 := result.(Model)

	if m2.current != screenExplore {
		t.Errorf("current = %d, want screenExplore (%d)", m2.current, screenExplore)
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd for window size forwarding")
	}
	// Execute the sizeCmd to cover the closure body
	sizeResult := cmd()
	if _, ok := sizeResult.(tea.WindowSizeMsg); !ok {
		t.Errorf("expected WindowSizeMsg, got %T", sizeResult)
	}
}

func TestModelMenuSelectionStats(t *testing.T) {
	m := NewModel(testServices(), testIdx())
	msg := curd.MenuSelectionMsg{Command: "stats"}

	result, _ := m.Update(msg)
	m2 := result.(Model)

	if m2.current != screenStats {
		t.Errorf("current = %d, want screenStats (%d)", m2.current, screenStats)
	}
}

func TestModelMenuSelectionStale(t *testing.T) {
	m := NewModel(testServices(), testIdx())
	msg := curd.MenuSelectionMsg{Command: "stale"}

	result, _ := m.Update(msg)
	m2 := result.(Model)

	if m2.current != screenStale {
		t.Errorf("current = %d, want screenStale (%d)", m2.current, screenStale)
	}
}

func TestModelMenuSelectionUnknown(t *testing.T) {
	m := NewModel(testServices(), testIdx())
	msg := curd.MenuSelectionMsg{Command: "unknown"}

	result, cmd := m.Update(msg)
	m2 := result.(Model)

	if m2.current != screenMenu {
		t.Errorf("current = %d, want screenMenu (%d)", m2.current, screenMenu)
	}
	if cmd != nil {
		t.Error("expected nil cmd for unknown command")
	}
}

func TestModelBackToMenu(t *testing.T) {
	m := NewModel(testServices(), testIdx())
	m.current = screenExplore
	m.explore = screens.NewExplore(m.services, m.nameIdx, 80, 24)

	msg := screens.BackToMenuMsg{}
	result, cmd := m.Update(msg)
	m2 := result.(Model)

	if m2.current != screenMenu {
		t.Errorf("current = %d, want screenMenu (%d)", m2.current, screenMenu)
	}
	if cmd == nil {
		t.Error("expected non-nil cmd (window size)")
	}
}

func TestModelGraphExportedMsg(t *testing.T) {
	m := NewModel(testServices(), testIdx())
	m.current = screenExplore
	m.explore = screens.NewExplore(m.services, m.nameIdx, 80, 24)

	msg := screens.GraphExportedMsg{Err: nil}
	result, cmd := m.Update(msg)
	_ = result

	if cmd != nil {
		t.Error("expected nil cmd from GraphExportedMsg")
	}
}

func TestModelViewAllScreens(t *testing.T) {
	svcs := testServices()
	idx := testIdx()

	tests := []struct {
		name   string
		screen screen
		setup  func(m *Model)
	}{
		{"menu", screenMenu, nil},
		{"explore", screenExplore, func(m *Model) {
			m.explore = screens.NewExplore(svcs, idx, 80, 24)
		}},
		{"stats", screenStats, func(m *Model) {
			m.stats = screens.NewStats(svcs, idx, 80, 24)
		}},
		{"stale", screenStale, func(m *Model) {
			m.stale = screens.NewStale(svcs, idx, 80, 24)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(svcs, idx)
			m.current = tt.screen
			if tt.setup != nil {
				tt.setup(&m)
			}

			v := m.View()
			if v.WindowTitle != "fondue" {
				t.Errorf("WindowTitle = %q, want %q", v.WindowTitle, "fondue")
			}
		})
	}
}

func TestModelDelegateToCurrentScreen(t *testing.T) {
	svcs := testServices()
	idx := testIdx()

	tests := []struct {
		name   string
		screen screen
		setup  func(m *Model)
	}{
		{"menu delegates", screenMenu, nil},
		{"explore delegates", screenExplore, func(m *Model) {
			m.explore = screens.NewExplore(svcs, idx, 80, 24)
		}},
		{"stats delegates", screenStats, func(m *Model) {
			m.stats = screens.NewStats(svcs, idx, 80, 24)
		}},
		{"stale delegates", screenStale, func(m *Model) {
			m.stale = screens.NewStale(svcs, idx, 80, 24)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(svcs, idx)
			m.current = tt.screen
			if tt.setup != nil {
				tt.setup(&m)
			}

			result, _ := m.Update(nil)
			if result == nil {
				t.Error("expected non-nil result from Update")
			}
		})
	}
}

func TestModelExportGraphMsg(t *testing.T) {
	svcs := testServices()
	idx := testIdx()
	m := NewModel(svcs, idx)
	m.current = screenExplore
	m.explore = screens.NewExplore(svcs, idx, 80, 24)

	msg := screens.ExportGraphMsg{Service: "auth-service", Hops: 1}
	result, cmd := m.Update(msg)
	m2 := result.(Model)

	if m2.current != screenExplore {
		t.Errorf("current = %d, want screenExplore", m2.current)
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd from ExportGraphMsg")
	}

	// Execute the command -- it calls ExportPNG which will fail without graphviz, but we test the flow
	resultMsg := cmd()
	if _, ok := resultMsg.(screens.GraphExportedMsg); !ok {
		t.Errorf("expected GraphExportedMsg, got %T", resultMsg)
	}
}

func TestModelGraphExportedMsgError(t *testing.T) {
	svcs := testServices()
	idx := testIdx()
	m := NewModel(svcs, idx)
	m.current = screenExplore
	m.explore = screens.NewExplore(svcs, idx, 80, 24)

	msg := screens.GraphExportedMsg{Err: fmt.Errorf("graphviz not installed")}
	result, cmd := m.Update(msg)
	_ = result

	if cmd != nil {
		t.Error("expected nil cmd from GraphExportedMsg with error")
	}
}

func TestModelUpdateQuitOnExplore(t *testing.T) {
	svcs := testServices()
	idx := testIdx()
	m := NewModel(svcs, idx)
	m.current = screenExplore
	m.explore = screens.NewExplore(svcs, idx, 80, 24)

	// q on explore should go back to menu via explore's handler, not quit
	msg := tea.KeyPressMsg{Code: 'q', Text: "q"}
	result, cmd := m.Update(msg)
	m2 := result.(Model)
	_ = m2
	// The explore handler should return a BackToMenuMsg cmd
	if cmd == nil {
		t.Error("expected non-nil cmd from q on explore")
	}
}

func TestModelUpdateKeypressOnStats(t *testing.T) {
	svcs := testServices()
	idx := testIdx()
	m := NewModel(svcs, idx)
	m.current = screenStats
	m.stats = screens.NewStats(svcs, idx, 80, 24)

	// Send a key that isn't ctrl+c or q-on-menu
	msg := tea.KeyPressMsg{Code: 'j', Text: "j"}
	result, _ := m.Update(msg)
	m2 := result.(Model)
	if m2.current != screenStats {
		t.Errorf("should stay on stats, got %d", m2.current)
	}
}

func TestModelUpdateKeypressOnStale(t *testing.T) {
	svcs := testServices()
	idx := testIdx()
	m := NewModel(svcs, idx)
	m.current = screenStale
	m.stale = screens.NewStale(svcs, idx, 80, 24)

	msg := tea.KeyPressMsg{Code: 'j', Text: "j"}
	result, _ := m.Update(msg)
	m2 := result.(Model)
	if m2.current != screenStale {
		t.Errorf("should stay on stale, got %d", m2.current)
	}
}

func TestScreenConstants(t *testing.T) {
	if screenMenu != 0 {
		t.Errorf("screenMenu = %d, want 0", screenMenu)
	}
	if screenExplore != 1 {
		t.Errorf("screenExplore = %d, want 1", screenExplore)
	}
	if screenStats != 2 {
		t.Errorf("screenStats = %d, want 2", screenStats)
	}
	if screenStale != 3 {
		t.Errorf("screenStale = %d, want 3", screenStale)
	}
}
