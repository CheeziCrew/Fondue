package graph

import (
	"strings"
	"testing"

	"github.com/CheeziCrew/fondue/scanner"
)

func TestImpactAnalysisNoConsumers(t *testing.T) {
	services := []scanner.Service{
		{Name: "a"},
		{Name: "b"},
	}
	idx := scanner.NewNameIndexFromServices(services)

	entries := ImpactAnalysis("a", services, idx)
	if len(entries) != 0 {
		t.Errorf("expected no impact, got %d entries", len(entries))
	}
}

func TestImpactAnalysisDirectConsumers(t *testing.T) {
	services := []scanner.Service{
		{Name: "a", DependedOnBy: []string{"b", "c"}},
		{Name: "b"},
		{Name: "c"},
	}
	idx := scanner.NewNameIndexFromServices(services)

	entries := ImpactAnalysis("a", services, idx)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	for _, e := range entries {
		if e.Depth != 1 {
			t.Errorf("expected depth 1, got %d for %s", e.Depth, e.Name)
		}
	}
}

func TestImpactAnalysisTransitive(t *testing.T) {
	services := []scanner.Service{
		{Name: "a", DependedOnBy: []string{"b"}},
		{Name: "b", DependedOnBy: []string{"c"}},
		{Name: "c", DependedOnBy: []string{"d"}},
		{Name: "d"},
	}
	idx := scanner.NewNameIndexFromServices(services)

	entries := ImpactAnalysis("a", services, idx)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	depths := make(map[string]int)
	for _, e := range entries {
		depths[e.Name] = e.Depth
	}
	if depths["b"] != 1 || depths["c"] != 2 || depths["d"] != 3 {
		t.Errorf("unexpected depths: %v", depths)
	}
}

func TestImpactAnalysisNoDuplicates(t *testing.T) {
	// Diamond: a is depended on by b and c, both depended on by d
	services := []scanner.Service{
		{Name: "a", DependedOnBy: []string{"b", "c"}},
		{Name: "b", DependedOnBy: []string{"d"}},
		{Name: "c", DependedOnBy: []string{"d"}},
		{Name: "d"},
	}
	idx := scanner.NewNameIndexFromServices(services)

	entries := ImpactAnalysis("a", services, idx)
	names := make(map[string]bool)
	for _, e := range entries {
		if names[e.Name] {
			t.Errorf("duplicate entry: %s", e.Name)
		}
		names[e.Name] = true
	}
	if len(entries) != 3 {
		t.Errorf("expected 3 unique entries, got %d", len(entries))
	}
}

func TestImpactAnalysisNonexistentRoot(t *testing.T) {
	services := []scanner.Service{{Name: "a"}}
	idx := scanner.NewNameIndexFromServices(services)

	entries := ImpactAnalysis("nonexistent", services, idx)
	if len(entries) != 0 {
		t.Errorf("expected no impact for nonexistent root, got %d", len(entries))
	}
}

func TestFormatImpactEmpty(t *testing.T) {
	got := FormatImpact("svc", nil)
	if !strings.Contains(got, "No downstream consumers") {
		t.Errorf("FormatImpact(nil) = %q, expected 'No downstream consumers'", got)
	}
}

func TestFormatImpactWithEntries(t *testing.T) {
	entries := []ImpactEntry{
		{Name: "b", Depth: 1},
		{Name: "c", Depth: 1},
		{Name: "d", Depth: 2},
	}
	got := FormatImpact("a", entries)
	if !strings.Contains(got, "3 service(s)") {
		t.Errorf("expected '3 service(s)' in output, got: %s", got)
	}
	if !strings.Contains(got, "Depth 1") || !strings.Contains(got, "Depth 2") {
		t.Errorf("expected depth headers in output, got: %s", got)
	}
}
