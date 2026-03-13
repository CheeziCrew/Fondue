package graph

import (
	"strings"
	"testing"

	"github.com/CheeziCrew/fondue/scanner"
)

func TestDetectCyclesNone(t *testing.T) {
	services := []scanner.Service{
		{Name: "a", Integrations: []scanner.Integration{{ClientID: "b"}}},
		{Name: "b", Integrations: []scanner.Integration{{ClientID: "c"}}},
		{Name: "c"},
	}
	idx := scanner.NewNameIndexFromServices(services)

	cycles := DetectCycles(services, idx)
	if len(cycles) != 0 {
		t.Errorf("expected no cycles, got %d", len(cycles))
	}
}

func TestDetectCyclesSimple(t *testing.T) {
	services := []scanner.Service{
		{Name: "a", Integrations: []scanner.Integration{{ClientID: "b"}}},
		{Name: "b", Integrations: []scanner.Integration{{ClientID: "a"}}},
	}
	idx := scanner.NewNameIndexFromServices(services)

	cycles := DetectCycles(services, idx)
	if len(cycles) != 1 {
		t.Fatalf("expected 1 cycle, got %d", len(cycles))
	}
	if len(cycles[0]) != 2 {
		t.Errorf("expected cycle of length 2, got %d", len(cycles[0]))
	}
}

func TestDetectCyclesTriangle(t *testing.T) {
	services := []scanner.Service{
		{Name: "a", Integrations: []scanner.Integration{{ClientID: "b"}}},
		{Name: "b", Integrations: []scanner.Integration{{ClientID: "c"}}},
		{Name: "c", Integrations: []scanner.Integration{{ClientID: "a"}}},
	}
	idx := scanner.NewNameIndexFromServices(services)

	cycles := DetectCycles(services, idx)
	if len(cycles) != 1 {
		t.Fatalf("expected 1 cycle, got %d", len(cycles))
	}
	if len(cycles[0]) != 3 {
		t.Errorf("expected cycle of length 3, got %d", len(cycles[0]))
	}
}

func TestDetectCyclesDedup(t *testing.T) {
	// A→B→A and B→A→B are the same cycle
	services := []scanner.Service{
		{Name: "a", Integrations: []scanner.Integration{{ClientID: "b"}}},
		{Name: "b", Integrations: []scanner.Integration{{ClientID: "a"}}},
	}
	idx := scanner.NewNameIndexFromServices(services)

	cycles := DetectCycles(services, idx)
	if len(cycles) != 1 {
		t.Errorf("expected dedup to 1 cycle, got %d", len(cycles))
	}
}

func TestDetectCyclesExternalIgnored(t *testing.T) {
	services := []scanner.Service{
		{Name: "a", Integrations: []scanner.Integration{{ClientID: "external-thing"}}},
	}
	idx := scanner.NewNameIndexFromServices(services)

	cycles := DetectCycles(services, idx)
	if len(cycles) != 0 {
		t.Errorf("expected no cycles with external dep, got %d", len(cycles))
	}
}

func TestDetectCyclesEmpty(t *testing.T) {
	cycles := DetectCycles(nil, scanner.NewNameIndexFromServices(nil))
	if len(cycles) != 0 {
		t.Errorf("expected no cycles for nil services, got %d", len(cycles))
	}
}

func TestDetectCyclesSelfLoop(t *testing.T) {
	services := []scanner.Service{
		{Name: "a", Integrations: []scanner.Integration{{ClientID: "a"}}},
	}
	idx := scanner.NewNameIndexFromServices(services)

	cycles := DetectCycles(services, idx)
	if len(cycles) != 1 {
		t.Fatalf("expected 1 self-loop cycle, got %d", len(cycles))
	}
	if len(cycles[0]) != 1 {
		t.Errorf("expected cycle of length 1 for self-loop, got %d", len(cycles[0]))
	}
}

func TestFormatCyclesEmpty(t *testing.T) {
	got := FormatCycles(nil)
	if got != "No cycles detected." {
		t.Errorf("FormatCycles(nil) = %q, want %q", got, "No cycles detected.")
	}
}

func TestFormatCycles(t *testing.T) {
	cycles := []Cycle{{"a", "b"}, {"x", "y", "z"}}
	got := FormatCycles(cycles)
	if !strings.Contains(got, "a → b → a") {
		t.Errorf("expected 'a → b → a' in output, got: %s", got)
	}
	if !strings.Contains(got, "x → y → z → x") {
		t.Errorf("expected 'x → y → z → x' in output, got: %s", got)
	}
}

func TestCanonicalKey(t *testing.T) {
	// Same cycle starting at different points should produce same key.
	key1 := canonicalKey(Cycle{"a", "b", "c"})
	key2 := canonicalKey(Cycle{"b", "c", "a"})
	key3 := canonicalKey(Cycle{"c", "a", "b"})
	if key1 != key2 || key2 != key3 {
		t.Errorf("expected same canonical key, got %q, %q, %q", key1, key2, key3)
	}
}

func TestCanonicalKeyEmpty(t *testing.T) {
	got := canonicalKey(nil)
	if got != "" {
		t.Errorf("canonicalKey(nil) = %q, want empty", got)
	}
}

func TestExtractCycle(t *testing.T) {
	path := []string{"a", "b", "c", "d"}
	cycle := extractCycle(path, "b")
	if len(cycle) != 3 || cycle[0] != "b" || cycle[1] != "c" || cycle[2] != "d" {
		t.Errorf("extractCycle() = %v, want [b c d]", cycle)
	}
}

func TestExtractCycleNotFound(t *testing.T) {
	path := []string{"a", "b"}
	cycle := extractCycle(path, "z")
	if cycle != nil {
		t.Errorf("extractCycle() = %v, want nil", cycle)
	}
}

func TestCycleString(t *testing.T) {
	got := cycleString(Cycle{"a", "b"})
	if got != "a → b → a" {
		t.Errorf("cycleString() = %q, want %q", got, "a → b → a")
	}
}
