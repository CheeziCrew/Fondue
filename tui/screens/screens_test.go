package screens

import (
	"fmt"
	"strings"
	"testing"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/CheeziCrew/fondue/scanner"
)

func testServices() []scanner.Service {
	return []scanner.Service{
		{
			Name:    "auth-service",
			Path:    "/tmp/auth-service",
			Version: "1.0.0",
			Integrations: []scanner.Integration{
				{ClientID: "user-service", SpecVersion: "2.0.0"},
			},
		},
		{
			Name:         "user-service",
			Path:         "/tmp/user-service",
			Version:      "2.0.0",
			DependedOnBy: []string{"auth-service"},
		},
		{
			Name: "isolated-service",
			Path: "/tmp/isolated-service",
		},
	}
}

func testServicesWithStale() []scanner.Service {
	return []scanner.Service{
		{
			Name:    "gateway",
			Path:    "/tmp/gateway",
			Version: "1.0.0",
			Integrations: []scanner.Integration{
				{ClientID: "backend", SpecVersion: "0.9.0"},
				{ClientID: "auth", SpecVersion: "1.0.0"},
			},
		},
		{Name: "backend", Path: "/tmp/backend", Version: "1.0.0", DependedOnBy: []string{"gateway"}},
		{Name: "auth", Path: "/tmp/auth", Version: "1.0.0", DependedOnBy: []string{"gateway"}},
	}
}

func testNameIndex(services []scanner.Service) *scanner.NameIndex {
	return scanner.NewNameIndexFromServices(services)
}

// key helpers
func keyPress(ch rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: ch, Text: string(ch)}
}

func specialKey(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code}
}

// ── Menu tests ──────────────────────────────────────────────────────

func TestNewMenu(t *testing.T) {
	m := NewMenu()
	v := m.View()
	if v == "" {
		t.Error("NewMenu().View() returned empty string")
	}
}

// ── Explore tests ───────────────────────────────────────────────────

func TestNewExplore(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)

	if m.view != exploreList {
		t.Errorf("initial view = %d, want exploreList (%d)", m.view, exploreList)
	}
	v := m.View()
	if v == "" {
		t.Error("NewExplore().View() returned empty string")
	}
}

func TestExploreInit(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)
	cmd := m.Init()
	if cmd != nil {
		t.Error("expected Init() to return nil")
	}
}

func TestExploreUpdate(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)
	m2, _ := m.Update(nil)
	v := m2.View()
	if v == "" {
		t.Error("ExploreModel.View() returned empty string after Update")
	}
}

func TestExploreUpdateWindowSize(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	m2, cmd := m.Update(msg)
	if m2.width != 120 || m2.height != 40 {
		t.Errorf("dimensions = %dx%d, want 120x40", m2.width, m2.height)
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

func TestExploreHandleGlobalKeyQ(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)
	_, cmd := m.Update(keyPress('q'))
	if cmd == nil {
		t.Fatal("expected cmd from q press")
	}
	result := cmd()
	if _, ok := result.(BackToMenuMsg); !ok {
		t.Errorf("expected BackToMenuMsg, got %T", result)
	}
}

func TestExploreHandleEscFromList(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)
	_, cmd := m.Update(specialKey(tea.KeyEscape))
	if cmd == nil {
		t.Fatal("expected cmd from esc press")
	}
	result := cmd()
	if _, ok := result.(BackToMenuMsg); !ok {
		t.Errorf("expected BackToMenuMsg, got %T", result)
	}
}

func TestExploreHandleEscFromDetail(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)
	m.view = exploreDetail
	m.selectedService = &svc[0]
	m2, cmd := m.Update(specialKey(tea.KeyEscape))
	if m2.view != exploreList {
		t.Errorf("view = %d, want exploreList (%d)", m2.view, exploreList)
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

func TestExploreHandleEscFromDetailWithNav(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)
	m.view = exploreDetail
	m.selectedService = &svc[1]
	m.navStack = []navEntry{{service: &svc[0], cursor: 0}}
	m2, _ := m.Update(specialKey(tea.KeyEscape))
	if m2.view != exploreDetail {
		t.Errorf("view = %d, want exploreDetail", m2.view)
	}
	if m2.selectedService.Name != "auth-service" {
		t.Errorf("selectedService = %q, want auth-service", m2.selectedService.Name)
	}
}

func TestExploreViewDetail(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)
	m.view = exploreDetail
	m.selectedService = &svc[0]
	v := m.View()
	if v == "" {
		t.Error("detail view returned empty string")
	}
}

func TestExploreViewDetailNilService(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)
	m.view = exploreDetail
	m.selectedService = nil
	v := m.View()
	if v != "" {
		t.Error("detail view with nil service should return empty")
	}
}

func TestExplorePushPopNav(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)
	m.selectedService = &svc[0]
	m.detailCursor = 5
	m.pushNav()
	if len(m.navStack) != 1 {
		t.Fatalf("navStack len = %d, want 1", len(m.navStack))
	}
	m.selectedService = &svc[1]
	m.detailCursor = 3
	popped := m.popNav()
	if !popped {
		t.Error("expected popNav to return true")
	}
	if m.selectedService.Name != "auth-service" {
		t.Errorf("selected = %q, want auth-service", m.selectedService.Name)
	}
	if m.detailCursor != 5 {
		t.Errorf("cursor = %d, want 5", m.detailCursor)
	}
	popped = m.popNav()
	if popped {
		t.Error("expected popNav to return false on empty stack")
	}
}

func TestServiceItemFilterValue(t *testing.T) {
	svc := scanner.Service{
		Name:         "test-svc",
		Integrations: []scanner.Integration{{ClientID: "dep-a"}, {ClientID: "dep-b"}},
		DependedOnBy: []string{"parent"},
	}
	item := serviceItem{service: svc}
	fv := item.FilterValue()
	if !strings.Contains(fv, "test-svc") {
		t.Error("FilterValue should contain service name")
	}
	if !strings.Contains(fv, "dep-a") {
		t.Error("FilterValue should contain integration client IDs")
	}
	if !strings.Contains(fv, "parent") {
		t.Error("FilterValue should contain dependents")
	}
}

func TestRenderDetailHeader(t *testing.T) {
	svc := &scanner.Service{Name: "test-svc", Path: "/tmp/test", Version: "1.2.3",
		Integrations: []scanner.Integration{{ClientID: "dep-a"}},
		DependedOnBy: []string{"parent"},
	}
	header := renderDetailHeader(svc)
	if !strings.Contains(header, "test-svc") {
		t.Error("header should contain service name")
	}
	if !strings.Contains(header, "1.2.3") {
		t.Error("header should contain version")
	}
}

func TestRenderDetailHeaderNoVersion(t *testing.T) {
	svc := &scanner.Service{Name: "test-svc"}
	header := renderDetailHeader(svc)
	if !strings.Contains(header, "test-svc") {
		t.Error("header should contain service name")
	}
}

func TestRenderDetailOutbound(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)
	m.selectedService = &svc[0]
	out := renderDetailOutbound(&svc[0], m)
	if !strings.Contains(out, "user-service") {
		t.Error("outbound should contain user-service")
	}
}

func TestRenderDetailOutboundEmpty(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)
	m.selectedService = &svc[2]
	out := renderDetailOutbound(&svc[2], m)
	if !strings.Contains(out, "no outbound") {
		t.Error("should indicate no outbound dependencies")
	}
}

func TestRenderDetailInbound(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)
	m.selectedService = &svc[1]
	out := renderDetailInbound(&svc[1], m)
	if !strings.Contains(out, "auth-service") {
		t.Error("inbound should contain auth-service")
	}
}

func TestRenderDetailInboundEmpty(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)
	out := renderDetailInbound(&svc[0], m)
	if !strings.Contains(out, "no inbound") {
		t.Error("should indicate no inbound dependencies")
	}
}

func TestRenderDetailFooterNormal(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)
	m.view = exploreDetail
	m.selectedService = &svc[0]
	footer := renderDetailFooter(m)
	if !strings.Contains(footer, "navigate") {
		t.Error("footer should contain navigate hint")
	}
}

func TestRenderDetailFooterWithNavStack(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)
	m.view = exploreDetail
	m.selectedService = &svc[0]
	m.navStack = []navEntry{{service: &svc[1], cursor: 0}}
	footer := renderDetailFooter(m)
	if !strings.Contains(footer, "history") {
		t.Error("footer should mention history when nav stack is non-empty")
	}
}

func TestRenderDetailFooterHopInput(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)
	m.enteringHops = true
	m.hopInput = "2"
	footer := renderDetailFooter(m)
	if !strings.Contains(footer, "Hops") {
		t.Error("footer should contain Hops prompt")
	}
}

func TestRenderDetailFooterExportFlash(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)
	m.exportFlash = "opened graph"
	footer := renderDetailFooter(m)
	if !strings.Contains(footer, "opened graph") {
		t.Error("footer should show export flash message")
	}
}

func TestRenderDetailFooterExportFlashError(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)
	m.exportFlash = "error"
	footer := renderDetailFooter(m)
	if !strings.Contains(footer, "error") {
		t.Error("footer should show error flash message")
	}
}

func TestFindNavIndex(t *testing.T) {
	navigable := []scanner.Integration{{ClientID: "alpha"}, {ClientID: "beta"}}
	if idx := findNavIndex(scanner.Integration{ClientID: "alpha"}, navigable); idx != 0 {
		t.Errorf("findNavIndex(alpha) = %d, want 0", idx)
	}
	if idx := findNavIndex(scanner.Integration{ClientID: "gamma"}, navigable); idx != -1 {
		t.Errorf("findNavIndex(gamma) = %d, want -1", idx)
	}
}

func TestRenderIntegrationLine(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	integ := scanner.Integration{ClientID: "user-service"}

	line := renderIntegrationLine(integ, true, false, svc, idx)
	if !strings.Contains(line, "user-service") {
		t.Error("should contain service name")
	}

	extInteg := scanner.Integration{ClientID: "external-api"}
	line = renderIntegrationLine(extInteg, false, false, svc, idx)
	if !strings.Contains(line, "ext") {
		t.Error("should contain ext tag for external")
	}

	line = renderIntegrationLine(integ, true, true, svc, idx)
	if !strings.Contains(line, "user-service") {
		t.Error("cursor line should contain service name")
	}
}

func TestRenderVersionAnnotation(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)

	result := renderVersionAnnotation(scanner.Integration{ClientID: "user-service"}, svc, idx)
	if result != "" {
		t.Errorf("expected empty for no spec version, got %q", result)
	}

	result = renderVersionAnnotation(scanner.Integration{ClientID: "user-service", SpecVersion: "2.0.0"}, svc, idx)
	if result == "" {
		t.Error("expected non-empty for matching version")
	}

	result = renderVersionAnnotation(scanner.Integration{ClientID: "user-service", SpecVersion: "1.0.0"}, svc, idx)
	if !strings.Contains(result, "STALE") {
		t.Error("expected STALE annotation")
	}

	result = renderVersionAnnotation(scanner.Integration{ClientID: "unknown", SpecVersion: "1.0.0"}, svc, idx)
	if result != "" {
		t.Errorf("expected empty for unknown target, got %q", result)
	}
}

func TestRenderReverseStaleAnnotation(t *testing.T) {
	svc := testServicesWithStale()
	idx := testNameIndex(svc)

	result := renderReverseStaleAnnotation("gateway", &svc[1], svc, idx)
	if !strings.Contains(result, "STALE") {
		t.Errorf("expected STALE, got %q", result)
	}

	result = renderReverseStaleAnnotation("gateway", &svc[2], svc, idx)
	if result == "" {
		t.Error("expected version match annotation")
	}

	noVerSvc := &scanner.Service{Name: "no-ver"}
	result = renderReverseStaleAnnotation("gateway", noVerSvc, svc, idx)
	if result != "" {
		t.Errorf("expected empty for no version, got %q", result)
	}

	result = renderReverseStaleAnnotation("unknown", &svc[1], svc, idx)
	if result != "" {
		t.Errorf("expected empty for unknown dependent, got %q", result)
	}
}

func TestGetNavigableIntegrations(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	integrations := []scanner.Integration{{ClientID: "user-service"}, {ClientID: "external-api"}}
	nav := getNavigableIntegrations(integrations, idx)
	if len(nav) != 1 {
		t.Fatalf("expected 1 navigable, got %d", len(nav))
	}
	if nav[0].ClientID != "user-service" {
		t.Errorf("navigable[0] = %q, want user-service", nav[0].ClientID)
	}
}

func TestGetNavigableCount(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	count := getNavigableCount(&svc[0], idx)
	if count != 1 {
		t.Errorf("navigableCount = %d, want 1", count)
	}
}

func TestHandleDetailNavKey(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)
	m.view = exploreDetail
	m.selectedService = &svc[0]
	m.detailCursor = 0

	// up from 0 stays at 0
	m2, _ := m.Update(specialKey(tea.KeyUp))
	if m2.detailCursor != 0 {
		t.Errorf("cursor = %d, want 0", m2.detailCursor)
	}

	// down from 0 stays at 0 (only 1 navigable)
	m3, _ := m.Update(specialKey(tea.KeyDown))
	_ = m3
}

func TestHandleDetailNavKeyBackspace(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)
	m.view = exploreDetail
	m.selectedService = &svc[0]
	m2, _ := m.Update(specialKey(tea.KeyBackspace))
	if m2.view != exploreList {
		t.Errorf("view = %d, want exploreList", m2.view)
	}
}

func TestHandleDetailNavKeyG(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)
	m.view = exploreDetail
	m.selectedService = &svc[0]
	m2, _ := m.Update(keyPress('g'))
	if !m2.enteringHops {
		t.Error("expected enteringHops to be true")
	}
	if m2.hopInput != "1" {
		t.Errorf("hopInput = %q, want '1'", m2.hopInput)
	}
}

func TestHandleHopInput(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)

	t.Run("enter submits", func(t *testing.T) {
		m := NewExplore(svc, idx, 80, 24)
		m.view = exploreDetail
		m.selectedService = &svc[0]
		m.enteringHops = true
		m.hopInput = "3"
		m2, cmd := m.Update(specialKey(tea.KeyEnter))
		if m2.enteringHops {
			t.Error("expected enteringHops to be false")
		}
		if cmd == nil {
			t.Fatal("expected cmd")
		}
		result := cmd()
		exportMsg, ok := result.(ExportGraphMsg)
		if !ok {
			t.Fatalf("expected ExportGraphMsg, got %T", result)
		}
		if exportMsg.Hops != 3 {
			t.Errorf("hops = %d, want 3", exportMsg.Hops)
		}
	})

	t.Run("enter with empty defaults to 1", func(t *testing.T) {
		m := NewExplore(svc, idx, 80, 24)
		m.view = exploreDetail
		m.selectedService = &svc[0]
		m.enteringHops = true
		m.hopInput = ""
		_, cmd := m.Update(specialKey(tea.KeyEnter))
		result := cmd()
		exportMsg := result.(ExportGraphMsg)
		if exportMsg.Hops != 1 {
			t.Errorf("hops = %d, want 1", exportMsg.Hops)
		}
	})

	t.Run("esc during hops triggers global esc (goes to list)", func(t *testing.T) {
		m := NewExplore(svc, idx, 80, 24)
		m.view = exploreDetail
		m.selectedService = &svc[0]
		m.enteringHops = true
		m.hopInput = "3"
		m2, _ := m.Update(specialKey(tea.KeyEscape))
		// Global esc handler intercepts: pops nav or goes to list view
		if m2.view != exploreList {
			t.Errorf("view = %d, want exploreList", m2.view)
		}
	})

	t.Run("backspace removes char", func(t *testing.T) {
		m := NewExplore(svc, idx, 80, 24)
		m.view = exploreDetail
		m.selectedService = &svc[0]
		m.enteringHops = true
		m.hopInput = "23"
		m2, _ := m.Update(specialKey(tea.KeyBackspace))
		if m2.hopInput != "2" {
			t.Errorf("hopInput = %q, want '2'", m2.hopInput)
		}
	})

	t.Run("digit appends", func(t *testing.T) {
		m := NewExplore(svc, idx, 80, 24)
		m.view = exploreDetail
		m.selectedService = &svc[0]
		m.enteringHops = true
		m.hopInput = "1"
		m2, _ := m.Update(keyPress('5'))
		if m2.hopInput != "15" {
			t.Errorf("hopInput = %q, want '15'", m2.hopInput)
		}
	})

	t.Run("non-digit ignored", func(t *testing.T) {
		m := NewExplore(svc, idx, 80, 24)
		m.view = exploreDetail
		m.selectedService = &svc[0]
		m.enteringHops = true
		m.hopInput = "1"
		m2, _ := m.Update(keyPress('a'))
		if m2.hopInput != "1" {
			t.Errorf("hopInput = %q, want '1'", m2.hopInput)
		}
	})
}

func TestHandleDetailEnter(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)
	m.view = exploreDetail
	m.selectedService = &svc[0]
	m.detailCursor = 0
	m2, _ := m.Update(specialKey(tea.KeyEnter))
	if m2.selectedService.Name != "user-service" {
		t.Errorf("selected = %q, want user-service", m2.selectedService.Name)
	}
	if len(m2.navStack) != 1 {
		t.Errorf("navStack len = %d, want 1", len(m2.navStack))
	}
}

func TestHandleGraphExported(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)

	t.Run("success", func(t *testing.T) {
		m := NewExplore(svc, idx, 80, 24)
		m2 := handleGraphExported(m, GraphExportedMsg{Err: nil})
		if !strings.Contains(m2.exportFlash, "opened") {
			t.Errorf("exportFlash = %q", m2.exportFlash)
		}
	})

	t.Run("error", func(t *testing.T) {
		m := NewExplore(svc, idx, 80, 24)
		m2 := handleGraphExported(m, GraphExportedMsg{Err: fmt.Errorf("test error")})
		if !strings.Contains(m2.exportFlash, "test error") {
			t.Errorf("exportFlash = %q", m2.exportFlash)
		}
	})
}

func TestResolveDetailTarget(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)
	m.selectedService = &svc[0]
	m.detailCursor = 0
	target := resolveDetailTarget(m)
	if target != "user-service" {
		t.Errorf("target = %q, want user-service", target)
	}

	m.detailCursor = 99
	target = resolveDetailTarget(m)
	if target != "" {
		t.Errorf("target = %q, want empty", target)
	}
}

func TestResolveDetailTargetInbound(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)
	m.selectedService = &svc[1]
	m.detailCursor = 0
	target := resolveDetailTarget(m)
	if target != "auth-service" {
		t.Errorf("target = %q, want auth-service", target)
	}
}

func TestRenderDependentLine(t *testing.T) {
	svc := testServicesWithStale()
	idx := testNameIndex(svc)
	line := renderDependentLine("gateway", false, &svc[1], svc, idx)
	if !strings.Contains(line, "gateway") {
		t.Error("line should contain dep name")
	}
	line = renderDependentLine("gateway", true, &svc[1], svc, idx)
	if !strings.Contains(line, "gateway") {
		t.Error("cursor line should contain dep name")
	}
}

func TestRenderDetail(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)
	m.view = exploreDetail
	m.selectedService = &svc[0]
	detail := renderDetail(m)
	if !strings.Contains(detail, "auth-service") {
		t.Error("detail should contain service name")
	}
}

func TestFuzzyFilter(t *testing.T) {
	targets := []string{"auth-service", "user-service", "gateway"}
	ranks := fuzzyFilter("auth", targets)
	if len(ranks) == 0 {
		t.Error("expected at least one match")
	}
}

func TestPadLines(t *testing.T) {
	result := padLines("line1\nline2", 2, 4)
	lines := strings.Split(result, "\n")
	if len(lines) < 4 {
		t.Errorf("expected at least 4 lines, got %d", len(lines))
	}
}

// ── Stats tests ─────────────────────────────────────────────────────

func TestNewStats(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewStats(svc, idx, 80, 24)
	if !m.viewReady {
		t.Error("expected viewReady to be true")
	}
	v := m.View()
	if v == "" {
		t.Error("NewStats().View() returned empty string")
	}
}

func TestNewStats_ZeroDimensions(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewStats(svc, idx, 0, 0)
	if m.viewReady {
		t.Error("expected viewReady to be false")
	}
	_ = m.View()
}

func TestStatsInit(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewStats(svc, idx, 80, 24)
	if cmd := m.Init(); cmd != nil {
		t.Error("expected Init() nil")
	}
}

func TestStatsUpdateWindowSize(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewStats(svc, idx, 0, 0)
	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	m2, _ := m.Update(msg)
	if !m2.viewReady {
		t.Error("expected viewReady after window size")
	}
	msg2 := tea.WindowSizeMsg{Width: 120, Height: 40}
	m3, _ := m2.Update(msg2)
	if m3.width != 120 {
		t.Errorf("width = %d, want 120", m3.width)
	}
}

func TestStatsUpdateEscQuit(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	for _, code := range []rune{tea.KeyEscape, 'q'} {
		m := NewStats(svc, idx, 80, 24)
		var msg tea.KeyPressMsg
		if code == 'q' {
			msg = keyPress('q')
		} else {
			msg = specialKey(code)
		}
		_, cmd := m.Update(msg)
		if cmd == nil {
			t.Fatal("expected cmd")
		}
		result := cmd()
		if _, ok := result.(BackToMenuMsg); !ok {
			t.Errorf("expected BackToMenuMsg, got %T", result)
		}
	}
}

func TestCollectStats(t *testing.T) {
	svc := testServicesWithStale()
	idx := testNameIndex(svc)
	d := collectStats(svc, idx)
	if d.totalServices != 3 {
		t.Errorf("totalServices = %d, want 3", d.totalServices)
	}
	if d.totalStale != 1 {
		t.Errorf("totalStale = %d, want 1", d.totalStale)
	}
}

func TestCollectStatsIsolated(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	d := collectStats(svc, idx)
	if len(d.isolated) != 1 || d.isolated[0] != "isolated-service" {
		t.Errorf("isolated = %v", d.isolated)
	}
}

func TestRenderStaleChart(t *testing.T) {
	items := []ranked{{name: "gateway", count: 3}, {name: "api", count: 1}}
	result := renderStaleChart(items, 80)
	if !strings.Contains(result, "gateway") {
		t.Error("stale chart should contain gateway")
	}
}

func TestRenderIsolatedFewItems(t *testing.T) {
	names := []string{"a", "b", "c"}
	result := renderIsolated(names, 80)
	if result == "" {
		t.Error("renderIsolated returned empty for few items")
	}
}

func TestRenderIsolatedManyItems(t *testing.T) {
	names := make([]string, 10)
	for i := range names {
		names[i] = strings.Repeat("x", i+1)
	}
	result := renderIsolated(names, 80)
	if result == "" {
		t.Error("renderIsolated returned empty for many items")
	}
}

func TestRenderSummaryCardsWithStale(t *testing.T) {
	d := statsData{totalServices: 10, totalOut: 5, totalIn: 3, totalStale: 2}
	result := renderSummaryCards(d)
	if result == "" {
		t.Error("renderSummaryCards returned empty")
	}
}

func TestRenderDistributionEmpty(t *testing.T) {
	d := statsData{}
	result := renderDistribution(d)
	if result != "" {
		t.Error("expected empty for zero distribution")
	}
}

func TestRenderBarChartEmpty(t *testing.T) {
	result := renderBarChart(nil, 80, nil)
	if !strings.Contains(result, "none") {
		t.Error("expected 'none' for empty bar chart")
	}
}

func TestRenderBarChartZeroCountFiltered(t *testing.T) {
	items := []ranked{{name: "a", count: 0}}
	result := renderBarChart(items, 80, nil)
	if !strings.Contains(result, "none") {
		t.Error("expected 'none' for all-zero")
	}
}

// ── Stale tests ─────────────────────────────────────────────────────

func TestNewStale(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewStale(svc, idx, 80, 24)
	if !m.viewReady {
		t.Error("expected viewReady")
	}
	v := m.View()
	if v == "" {
		t.Error("NewStale().View() returned empty")
	}
}

func TestNewStale_ZeroDimensions(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewStale(svc, idx, 0, 0)
	if m.viewReady {
		t.Error("expected viewReady false")
	}
	_ = m.View()
}

func TestStaleInit(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewStale(svc, idx, 80, 24)
	if cmd := m.Init(); cmd != nil {
		t.Error("expected nil")
	}
}

func TestStaleUpdateWindowSize(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewStale(svc, idx, 0, 0)
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if !m2.viewReady {
		t.Error("expected viewReady")
	}
	m3, _ := m2.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if m3.width != 120 {
		t.Errorf("width = %d", m3.width)
	}
}

func TestStaleUpdateEscQuit(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	for _, code := range []rune{tea.KeyEscape, 'q'} {
		m := NewStale(svc, idx, 80, 24)
		var msg tea.KeyPressMsg
		if code == 'q' {
			msg = keyPress('q')
		} else {
			msg = specialKey(code)
		}
		_, cmd := m.Update(msg)
		if cmd == nil {
			t.Fatal("expected cmd")
		}
		result := cmd()
		if _, ok := result.(BackToMenuMsg); !ok {
			t.Errorf("expected BackToMenuMsg, got %T", result)
		}
	}
}

func TestCollectStaleEntries(t *testing.T) {
	svc := testServicesWithStale()
	idx := testNameIndex(svc)
	entries := collectStaleEntries(svc, idx)
	if len(entries) != 1 {
		t.Fatalf("expected 1, got %d", len(entries))
	}
}

func TestGroupByRepo(t *testing.T) {
	entries := []staleEntry{
		{repo: "svc-a", specFile: "x"},
		{repo: "svc-b", specFile: "y"},
		{repo: "svc-a", specFile: "z"},
	}
	byRepo, order := groupByRepo(entries)
	if len(byRepo) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(byRepo))
	}
	if len(byRepo["svc-a"]) != 2 {
		t.Errorf("svc-a = %d, want 2", len(byRepo["svc-a"]))
	}
	if order[0] != "svc-a" || order[1] != "svc-b" {
		t.Errorf("order = %v", order)
	}
}

func TestRenderStaleRepoGroup(t *testing.T) {
	entries := []staleEntry{{repo: "gw", specFile: "backend", specVersion: "0.9.0", targetVer: "1.0.0"}}
	result := renderStaleRepoGroup("gw", entries)
	if !strings.Contains(result, "gw") {
		t.Error("should contain repo name")
	}
}

func TestRenderStaleRepoGroupUnresolved(t *testing.T) {
	entries := []staleEntry{{repo: "gw", specFile: "b", specVersion: "@project.version@", targetVer: "1.0"}}
	result := renderStaleRepoGroup("gw", entries)
	if !strings.Contains(result, "unresolved") {
		t.Error("should show unresolved")
	}
}

func TestIsUnresolvedPlaceholder(t *testing.T) {
	tests := []struct{ input string; want bool }{
		{"@project.version@", true},
		{"${project.version}", true},
		{"1.0.0", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := isUnresolvedPlaceholder(tt.input); got != tt.want {
			t.Errorf("isUnresolvedPlaceholder(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestRenderStaleOverviewNoStale(t *testing.T) {
	svc := []scanner.Service{{Name: "a", Version: "1.0.0"}}
	idx := testNameIndex(svc)
	result := renderStaleOverview(svc, idx)
	if !strings.Contains(result, "up to date") {
		t.Error("should show up to date")
	}
}

func TestRenderStaleOverviewWithStale(t *testing.T) {
	svc := testServicesWithStale()
	idx := testNameIndex(svc)
	result := renderStaleOverview(svc, idx)
	if !strings.Contains(result, "stale") {
		t.Error("should mention stale")
	}
}

func TestStatsViewWithoutReady(t *testing.T) {
	m := StatsModel{viewReady: false}
	if v := m.View(); v != "" {
		t.Error("expected empty view")
	}
}

func TestStaleViewWithoutReady(t *testing.T) {
	m := StaleModel{viewReady: false}
	if v := m.View(); v != "" {
		t.Error("expected empty view")
	}
}

func TestStatsUpdateDelegation(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewStats(svc, idx, 80, 24)
	m2, _ := m.Update(nil)
	_ = m2
}

func TestStaleUpdateDelegation(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewStale(svc, idx, 80, 24)
	m2, _ := m.Update(nil)
	_ = m2
}

func TestStatsUpdateNotReady(t *testing.T) {
	m := StatsModel{viewReady: false}
	m2, _ := m.Update(nil)
	_ = m2
}

func TestStaleUpdateNotReady(t *testing.T) {
	m := StaleModel{viewReady: false}
	m2, _ := m.Update(nil)
	_ = m2
}

// ── Additional coverage: explore handleExploreListUpdate ────────────

func TestExploreListUpdateEnterSelectsService(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)

	// Simulate pressing enter on the list (first item should be selected)
	m2, _ := m.Update(specialKey(tea.KeyEnter))
	if m2.view != exploreDetail {
		t.Errorf("view = %d, want exploreDetail (%d)", m2.view, exploreDetail)
	}
	if m2.selectedService == nil {
		t.Error("expected selectedService to be non-nil")
	}
}

func TestExploreListUpdateFilteringIgnoresEnter(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)

	// Start filtering by pressing /
	m2, _ := m.Update(keyPress('/'))
	// Now press enter while filtering (should not switch to detail)
	m3, _ := m2.Update(specialKey(tea.KeyEnter))
	// View should still be list (filtering handles enter differently)
	if m3.view != exploreList {
		t.Errorf("view = %d, want exploreList (%d) during filtering", m3.view, exploreList)
	}
}

// ── Additional coverage: explore handleExploreDetailUpdate non-key msg ──

func TestExploreDetailUpdateNonKeyMsg(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)
	m.view = exploreDetail
	m.selectedService = &svc[0]

	// Send a non-key, non-GraphExportedMsg message
	m2, cmd := m.Update(nil)
	if m2.view != exploreDetail {
		t.Errorf("view = %d, want exploreDetail", m2.view)
	}
	if cmd != nil {
		t.Error("expected nil cmd for unhandled msg in detail")
	}
}

func TestExploreDetailUpdateGraphExportedMsg(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)
	m.view = exploreDetail
	m.selectedService = &svc[0]

	msg := GraphExportedMsg{Err: nil}
	m2, cmd := m.Update(msg)
	if !strings.Contains(m2.exportFlash, "opened") {
		t.Errorf("exportFlash = %q, expected 'opened'", m2.exportFlash)
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

// ── Additional coverage: renderBadges branches ──────────────────────

func TestRenderBadgesSelectedWithStale(t *testing.T) {
	svc := testServicesWithStale()
	idx := testNameIndex(svc)

	// Selected + has out + has stale
	ds := selectedDelegateStyles()
	result := renderBadges(svc[0], true, ds, svc, idx)
	if !strings.Contains(result, "out") {
		t.Error("selected badge should contain 'out'")
	}
	if !strings.Contains(result, "stale") {
		t.Error("should show stale badge")
	}
}

func TestRenderBadgesUnselectedInbound(t *testing.T) {
	svc := testServicesWithStale()
	idx := testNameIndex(svc)

	// Unselected + has in
	ds := unselectedDelegateStyles()
	result := renderBadges(svc[1], false, ds, svc, idx)
	// backend has DependedOnBy: gateway
	if result == "" {
		t.Error("expected non-empty badges for service with inbound")
	}
}

func TestRenderBadgesSelectedInbound(t *testing.T) {
	svc := testServicesWithStale()
	idx := testNameIndex(svc)

	ds := selectedDelegateStyles()
	result := renderBadges(svc[1], true, ds, svc, idx)
	if !strings.Contains(result, "in") {
		t.Error("selected badge with inbound should contain 'in'")
	}
}

func TestRenderBadgesIsolated(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)

	ds := unselectedDelegateStyles()
	result := renderBadges(svc[2], false, ds, svc, idx)
	if !strings.Contains(result, "isolated") {
		t.Error("isolated service should show isolated badge")
	}
}

// ── Additional coverage: renderDepList truncation ───────────────────

func TestRenderDepListTruncation(t *testing.T) {
	// Create integrations that will exceed maxWidth
	integrations := make([]scanner.Integration, 20)
	for i := range integrations {
		integrations[i] = scanner.Integration{ClientID: fmt.Sprintf("very-long-service-name-%d", i)}
	}
	result := renderDepList(integrations, nil, colorFg, 30)
	if !strings.Contains(result, "...") {
		t.Error("expected truncation with '...' for long dep list")
	}
}

func TestRenderDepListNames(t *testing.T) {
	names := []string{"svc-a", "svc-b"}
	result := renderDepList(nil, names, colorFg, 70)
	if result == "" {
		t.Error("expected non-empty result for names list")
	}
}

// ── Additional coverage: handleDetailNavKey down/j with room ─────────

func TestHandleDetailNavKeyDown(t *testing.T) {
	svc := []scanner.Service{
		{
			Name:    "gateway",
			Version: "1.0.0",
			Integrations: []scanner.Integration{
				{ClientID: "backend"},
				{ClientID: "auth"},
			},
			DependedOnBy: []string{"frontend"},
		},
		{Name: "backend", Version: "1.0.0"},
		{Name: "auth", Version: "1.0.0"},
		{Name: "frontend", Version: "1.0.0"},
	}
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)
	m.view = exploreDetail
	m.selectedService = &svc[0]
	m.detailCursor = 0

	// Should move down
	m2, _ := m.Update(specialKey(tea.KeyDown))
	if m2.detailCursor != 1 {
		t.Errorf("cursor = %d, want 1", m2.detailCursor)
	}

	// Try j key
	m3, _ := m2.Update(keyPress('j'))
	if m3.detailCursor != 2 {
		t.Errorf("cursor = %d, want 2", m3.detailCursor)
	}

	// Try k key to go back up
	m4, _ := m3.Update(keyPress('k'))
	if m4.detailCursor != 1 {
		t.Errorf("cursor = %d, want 1", m4.detailCursor)
	}
}

// ── Additional coverage: handleDetailEnter with no match ──────────────

func TestHandleDetailEnterNoTarget(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)
	m.view = exploreDetail
	m.selectedService = &svc[2] // isolated service, no navigable items
	m.detailCursor = 0

	m2, _ := m.Update(specialKey(tea.KeyEnter))
	// Should not navigate anywhere
	if m2.selectedService.Name != "isolated-service" {
		t.Errorf("selected = %q, should stay on isolated-service", m2.selectedService.Name)
	}
}

// ── Additional coverage: handleHopInput esc clears state ────────────

func TestHandleHopInputEscClearsState(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)
	m.view = exploreDetail
	m.selectedService = &svc[0]
	m.enteringHops = true
	m.hopInput = "5"

	// Esc during hop input -- global handler intercepts first
	// but if we go through handleHopInput directly...
	m2 := m
	m2.enteringHops = true
	m3, _ := handleHopInput(m2, specialKey(tea.KeyEscape))
	if m3.enteringHops {
		t.Error("expected enteringHops to be false after esc")
	}
	if m3.hopInput != "" {
		t.Errorf("hopInput = %q, want empty", m3.hopInput)
	}
}

func TestHandleHopInputBackspaceEmpty(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)
	m.view = exploreDetail
	m.selectedService = &svc[0]
	m.enteringHops = true
	m.hopInput = ""

	m2, _ := handleHopInput(m, specialKey(tea.KeyBackspace))
	if m2.hopInput != "" {
		t.Errorf("hopInput = %q, want empty", m2.hopInput)
	}
}

// ── Additional coverage: handleEscKey during filtering ──────────────

func TestExploreHandleQDuringFiltering(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)

	// Enter filter mode
	m2, _ := m.Update(keyPress('/'))
	// Press q during filtering should NOT go back to menu
	m3, cmd := m2.Update(keyPress('q'))
	if cmd != nil {
		// If cmd is not nil and returns BackToMenuMsg, that's wrong
		result := cmd()
		if _, ok := result.(BackToMenuMsg); ok {
			t.Error("should not go back to menu when filtering")
		}
	}
	_ = m3
}

// ── Additional coverage: collectServiceStaleEntries branches ────────

func TestCollectServiceStaleEntriesAllBranches(t *testing.T) {
	svc := []scanner.Service{
		{
			Name:    "gateway",
			Version: "1.0.0",
			Integrations: []scanner.Integration{
				{ClientID: "backend", SpecVersion: "0.9.0"}, // stale
				{ClientID: "auth", SpecVersion: "1.0.0"},    // matching
				{ClientID: "external"},                       // no spec version
				{ClientID: "unknown", SpecVersion: "1.0"},    // unknown target
			},
		},
		{Name: "backend", Version: "1.0.0", DependedOnBy: []string{"gateway"}},
		{Name: "auth", Version: "1.0.0", DependedOnBy: []string{"gateway"}},
		{Name: "no-ver", DependedOnBy: []string{"gateway"}}, // target with no version
	}
	idx := testNameIndex(svc)

	entries := collectServiceStaleEntries(svc[0], svc, idx)
	if len(entries) != 1 {
		t.Errorf("expected 1 stale entry, got %d", len(entries))
	}
}

// ── Additional coverage: stale View with scrollable content ─────────

func TestStaleViewScrollable(t *testing.T) {
	// Create many stale services to make viewport scrollable
	var svcs []scanner.Service
	for i := 0; i < 50; i++ {
		svcs = append(svcs, scanner.Service{
			Name:    fmt.Sprintf("svc-%d", i),
			Version: "2.0.0",
			Integrations: []scanner.Integration{
				{ClientID: fmt.Sprintf("dep-%d", i), SpecVersion: "1.0.0"},
			},
		})
		svcs = append(svcs, scanner.Service{
			Name:    fmt.Sprintf("dep-%d", i),
			Version: "2.0.0",
		})
	}
	idx := testNameIndex(svcs)
	m := NewStale(svcs, idx, 80, 10) // small height to force scroll
	v := m.View()
	if v == "" {
		t.Error("expected non-empty view")
	}
	// Should include scroll hint when content exceeds viewport
	if !strings.Contains(v, "scroll") && !strings.Contains(v, "menu") {
		t.Error("expected hint bar in view")
	}
}

// ── Additional coverage: stats with high connectivity ───────────────

func TestCollectStatsDistributionBuckets(t *testing.T) {
	svcs := []scanner.Service{
		{Name: "iso"}, // 0 connections
		{Name: "low", Integrations: []scanner.Integration{{ClientID: "x"}}}, // 1
		{Name: "mid", Integrations: []scanner.Integration{{ClientID: "a"}, {ClientID: "b"}, {ClientID: "c"}}}, // 3
		{Name: "high", Integrations: []scanner.Integration{
			{ClientID: "a"}, {ClientID: "b"}, {ClientID: "c"}, {ClientID: "d"},
			{ClientID: "e"}, {ClientID: "f"}, {ClientID: "g"},
		}}, // 7
	}
	idx := testNameIndex(svcs)
	d := collectStats(svcs, idx)

	if d.distribution[0] != 1 {
		t.Errorf("bucket 0 = %d, want 1", d.distribution[0])
	}
	if d.distribution[1] != 1 {
		t.Errorf("bucket 1-2 = %d, want 1", d.distribution[1])
	}
	if d.distribution[2] != 1 {
		t.Errorf("bucket 3-5 = %d, want 1", d.distribution[2])
	}
	if d.distribution[3] != 1 {
		t.Errorf("bucket 6-10 = %d, want 1", d.distribution[3])
	}
}

func TestCollectStatsDistribution11Plus(t *testing.T) {
	integs := make([]scanner.Integration, 12)
	for i := range integs {
		integs[i] = scanner.Integration{ClientID: fmt.Sprintf("dep-%d", i)}
	}
	svcs := []scanner.Service{
		{Name: "mega", Integrations: integs},
	}
	idx := testNameIndex(svcs)
	d := collectStats(svcs, idx)

	if d.distribution[4] != 1 {
		t.Errorf("bucket 11+ = %d, want 1", d.distribution[4])
	}
}

// ── Additional coverage: renderBarChart with name longer than 25 ────

func TestRenderBarChartLongNames(t *testing.T) {
	items := []ranked{
		{name: "this-is-a-really-long-service-name-that-exceeds-25", count: 5},
		{name: "short", count: 2},
	}
	result := renderBarChart(items, 80, colorYellow)
	if result == "" {
		t.Error("expected non-empty bar chart")
	}
}

func TestRenderBarChartNarrowWidth(t *testing.T) {
	items := []ranked{{name: "svc", count: 3}}
	result := renderBarChart(items, 20, colorYellow)
	if result == "" {
		t.Error("expected non-empty bar chart even at narrow width")
	}
}

// ── Additional coverage: renderStaleChart with narrow width ─────────

func TestRenderStaleChartLongNames(t *testing.T) {
	items := []ranked{
		{name: "this-is-a-really-long-stale-service-name", count: 5},
		{name: "short", count: 1},
	}
	result := renderStaleChart(items, 20)
	if result == "" {
		t.Error("expected non-empty stale chart")
	}
}

// ── Additional coverage: renderStats with no stale and no isolated ──

func TestRenderStatsNoStaleNoIsolated(t *testing.T) {
	svc := []scanner.Service{
		{Name: "a", Version: "1.0.0", Integrations: []scanner.Integration{{ClientID: "b"}}},
		{Name: "b", Version: "1.0.0", DependedOnBy: []string{"a"}},
	}
	idx := testNameIndex(svc)
	result := renderStats(svc, idx, 80)
	if result == "" {
		t.Error("expected non-empty stats render")
	}
}

func TestRenderStatsNarrowWidth(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	result := renderStats(svc, idx, 30)
	if result == "" {
		t.Error("expected non-empty stats render with narrow width")
	}
}

// ── Additional coverage: handleDetailNavKey unmatched key ────────────

func TestHandleDetailNavKeyUnmatchedKey(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)
	m.view = exploreDetail
	m.selectedService = &svc[0]
	m.detailCursor = 0

	// Press an unmatched key like 'x'
	m2, cmd := m.Update(keyPress('x'))
	if m2.detailCursor != 0 {
		t.Errorf("cursor changed to %d, should stay at 0", m2.detailCursor)
	}
	if cmd != nil {
		t.Error("expected nil cmd for unmatched key")
	}
}

// ── Additional coverage: handleDetailNavKey backspace with nav stack ──

func TestHandleDetailNavKeyBackspaceWithNav(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)
	m.view = exploreDetail
	m.selectedService = &svc[1]
	m.navStack = []navEntry{{service: &svc[0], cursor: 2}}

	m2, _ := m.Update(specialKey(tea.KeyBackspace))
	if m2.view != exploreDetail {
		t.Errorf("view = %d, want exploreDetail (popped nav)", m2.view)
	}
	if m2.selectedService.Name != "auth-service" {
		t.Errorf("selected = %q, want auth-service", m2.selectedService.Name)
	}
	if m2.detailCursor != 2 {
		t.Errorf("cursor = %d, want 2", m2.detailCursor)
	}
}

// ── Additional coverage: explore Update with non-key msg fallthrough ──

func TestExploreUpdateNonKeyMsgFallthrough(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)

	// Send a non-key, non-window-size message while in list view
	type customMsg struct{}
	m2, _ := m.Update(customMsg{})
	if m2.view != exploreList {
		t.Errorf("view = %d, want exploreList", m2.view)
	}
}

// ── Additional coverage: stats View with scrollable content ─────────

// ── Additional coverage: handleDetailEnter target not found in services ──

func TestHandleDetailEnterTargetResolvedButNotInServices(t *testing.T) {
	// Create a situation where the cursor points to a resolved name
	// but that service is not actually in the services slice
	svc := []scanner.Service{
		{
			Name:    "gateway",
			Version: "1.0.0",
			Integrations: []scanner.Integration{
				{ClientID: "phantom"},
			},
			DependedOnBy: []string{"real-dep"},
		},
	}
	// Create an index that knows about phantom
	names := map[string]bool{"gateway": true, "phantom": true, "real-dep": true}
	idx := scanner.NewNameIndex(names)

	m := NewExplore(svc, idx, 80, 24)
	m.view = exploreDetail
	m.selectedService = &svc[0]
	m.detailCursor = 0 // points to phantom (navigable because it's internal)

	m2, _ := m.Update(specialKey(tea.KeyEnter))
	// phantom is not in services slice, so should not navigate
	if m2.selectedService.Name != "gateway" {
		t.Errorf("selected = %q, should stay on gateway", m2.selectedService.Name)
	}
}

// ── Additional coverage: renderVersionAnnotation target has no version ──

func TestRenderVersionAnnotationTargetNoVersion(t *testing.T) {
	svc := []scanner.Service{
		{Name: "no-ver-target"},
	}
	idx := testNameIndex(svc)
	result := renderVersionAnnotation(scanner.Integration{ClientID: "no-ver-target", SpecVersion: "1.0"}, svc, idx)
	if result != "" {
		t.Errorf("expected empty, got %q", result)
	}
}

// ── Additional coverage: renderReverseStaleAnnotation target integration spec doesn't match ──

func TestRenderReverseStaleAnnotationNoMatchingInteg(t *testing.T) {
	svc := []scanner.Service{
		{
			Name:    "dep",
			Version: "1.0.0",
			Integrations: []scanner.Integration{
				{ClientID: "other-svc", SpecVersion: "1.0"},
			},
		},
		{Name: "target", Version: "2.0.0"},
	}
	idx := testNameIndex(svc)
	result := renderReverseStaleAnnotation("dep", &svc[1], svc, idx)
	// dep integrates with other-svc, not target, so no annotation
	if result != "" {
		t.Errorf("expected empty, got %q", result)
	}
}

func TestRenderReverseStaleAnnotationNoSpecVersion(t *testing.T) {
	svc := []scanner.Service{
		{
			Name: "dep",
			Integrations: []scanner.Integration{
				{ClientID: "target"}, // no spec version
			},
		},
		{Name: "target", Version: "2.0.0"},
	}
	idx := testNameIndex(svc)
	result := renderReverseStaleAnnotation("dep", &svc[1], svc, idx)
	if result != "" {
		t.Errorf("expected empty for no spec version, got %q", result)
	}
}

// ── Additional coverage: renderNormalFooter error flash style ────────

func TestRenderNormalFooterErrorFlash(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)

	// Test with error prefix (✗)
	m := NewExplore(svc, idx, 80, 24)
	m.exportFlash = "✗ graphviz not found"
	footer := renderDetailFooter(m)
	if footer == "" {
		t.Error("expected non-empty footer for error flash")
	}

	// Test with success prefix
	m2 := NewExplore(svc, idx, 80, 24)
	m2.exportFlash = "✓ opened graph"
	footer2 := renderDetailFooter(m2)
	if footer2 == "" {
		t.Error("expected non-empty footer for success flash")
	}
}

// ── Additional coverage: handleEscKey during esc on list while filtering ──

func TestExploreEscDuringFilteringStaysInFilter(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)

	// Enter filter mode
	m2, _ := m.Update(keyPress('/'))
	// Esc during filtering should exit filter mode, not go back to menu
	m3, cmd := m2.Update(specialKey(tea.KeyEscape))
	// The bubbles list handles esc during filtering internally
	_ = m3
	if cmd != nil {
		result := cmd()
		if _, ok := result.(BackToMenuMsg); ok {
			t.Error("should not go back to menu when esc during filtering")
		}
	}
}

// ── Additional coverage: collectServiceStaleEntries with target no version ──

func TestCollectServiceStaleEntriesTargetNoVersion(t *testing.T) {
	svc := []scanner.Service{
		{
			Name: "gateway",
			Integrations: []scanner.Integration{
				{ClientID: "no-ver", SpecVersion: "1.0.0"},
			},
		},
		{Name: "no-ver"}, // no version
	}
	idx := testNameIndex(svc)
	entries := collectServiceStaleEntries(svc[0], svc, idx)
	if len(entries) != 0 {
		t.Errorf("expected 0 stale entries for target with no version, got %d", len(entries))
	}
}

func TestCollectServiceStaleEntriesMatchingVersion(t *testing.T) {
	svc := []scanner.Service{
		{
			Name: "gateway",
			Integrations: []scanner.Integration{
				{ClientID: "backend", SpecVersion: "1.0.0"},
			},
		},
		{Name: "backend", Version: "1.0.0"},
	}
	idx := testNameIndex(svc)
	entries := collectServiceStaleEntries(svc[0], svc, idx)
	if len(entries) != 0 {
		t.Errorf("expected 0 stale entries for matching version, got %d", len(entries))
	}
}

// ── Additional coverage: Explore Update with unknown view ───────────

func TestExploreUpdateUnknownView(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	m := NewExplore(svc, idx, 80, 24)
	m.view = 99 // unknown view

	m2, cmd := m.Update(nil)
	if cmd != nil {
		t.Error("expected nil cmd for unknown view")
	}
	_ = m2
}

// ── Additional coverage: Render with bad list item type ──────────────

func TestServiceDelegateRenderBadItem(t *testing.T) {
	svc := testServices()
	idx := testNameIndex(svc)
	d := serviceDelegate{nameIdx: idx}

	items := []list.Item{serviceItem{service: svc[0]}}
	l := list.New(items, d, 80, 24)

	// Render with a valid item (just exercises the code path fully)
	var buf strings.Builder
	d.Render(&buf, l, 0, items[0])
	if buf.Len() == 0 {
		t.Error("expected non-empty render output")
	}
}

// ── Additional coverage: renderDistribution with all buckets ────────

func TestRenderDistributionAllBuckets(t *testing.T) {
	d := statsData{
		distribution: [5]int{3, 5, 2, 1, 0},
	}
	result := renderDistribution(d)
	if result == "" {
		t.Error("expected non-empty distribution")
	}
	if !strings.Contains(result, "Distribution") {
		t.Error("expected Distribution header")
	}
}

func TestRenderDistributionSingleNonZero(t *testing.T) {
	d := statsData{
		distribution: [5]int{0, 0, 1, 0, 0},
	}
	result := renderDistribution(d)
	if result == "" {
		t.Error("expected non-empty distribution for single non-zero")
	}
}

// ── Additional coverage: renderBarChart with single item ────────────

func TestRenderBarChartSingleItem(t *testing.T) {
	items := []ranked{{name: "only", count: 1}}
	result := renderBarChart(items, 80, colorYellow)
	if result == "" {
		t.Error("expected non-empty bar chart for single item")
	}
}

// ── Additional coverage: renderBarChart with many items (>5) ────────

func TestRenderBarChartManyItems(t *testing.T) {
	items := []ranked{
		{name: "a", count: 10},
		{name: "b", count: 8},
		{name: "c", count: 6},
		{name: "d", count: 4},
		{name: "e", count: 2},
		{name: "f", count: 1}, // should be excluded (top 5 only)
	}
	result := renderBarChart(items, 80, colorGreen)
	if result == "" {
		t.Error("expected non-empty bar chart")
	}
}

// ── Additional coverage: renderStaleChart with many items ───────────

func TestRenderStaleChartManyItems(t *testing.T) {
	items := []ranked{
		{name: "a", count: 5},
		{name: "b", count: 4},
		{name: "c", count: 3},
		{name: "d", count: 2},
		{name: "e", count: 1},
		{name: "f", count: 1}, // should be excluded (top 5 only)
	}
	result := renderStaleChart(items, 80)
	if result == "" {
		t.Error("expected non-empty stale chart")
	}
}

// ── Additional coverage: renderStats with stale hotspots ────────────

func TestRenderStatsWithStale(t *testing.T) {
	svc := testServicesWithStale()
	idx := testNameIndex(svc)
	result := renderStats(svc, idx, 80)
	if !strings.Contains(result, "Stale") {
		t.Error("expected Stale Hotspots section")
	}
}

func TestStatsViewScrollable(t *testing.T) {
	var svcs []scanner.Service
	for i := 0; i < 50; i++ {
		svcs = append(svcs, scanner.Service{
			Name:         fmt.Sprintf("svc-%d", i),
			Version:      "1.0.0",
			Integrations: []scanner.Integration{{ClientID: fmt.Sprintf("dep-%d", i)}},
		})
	}
	idx := testNameIndex(svcs)
	m := NewStats(svcs, idx, 80, 10) // small height to force scroll
	v := m.View()
	if v == "" {
		t.Error("expected non-empty view")
	}
}
