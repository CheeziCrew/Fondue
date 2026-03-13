package graph

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/CheeziCrew/fondue/scanner"
)

// ── HeatColor tests ─────────────────────────────────────────────────

func TestHeatColorBoundaries(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{0.0, "#4285f4"},  // first stop: blue
		{1.0, "#ef4444"},  // last stop: red
		{-0.5, "#4285f4"}, // clamped to 0
		{1.5, "#ef4444"},  // clamped to 1
	}

	for _, tt := range tests {
		got := HeatColor(tt.input)
		if got != tt.want {
			t.Errorf("HeatColor(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestHeatColorMidpoints(t *testing.T) {
	// At 0.5 we should be between the second and third stops.
	color := HeatColor(0.5)
	if !strings.HasPrefix(color, "#") || len(color) != 7 {
		t.Errorf("HeatColor(0.5) = %q, want valid hex color", color)
	}

	// Ensure monotonic progression: colors at different intensities should differ.
	c1 := HeatColor(0.0)
	c2 := HeatColor(0.5)
	c3 := HeatColor(1.0)
	if c1 == c2 || c2 == c3 {
		t.Errorf("expected different colors at 0.0, 0.5, 1.0 but got %q, %q, %q", c1, c2, c3)
	}
}

// ── maxDepCount tests ───────────────────────────────────────────────

func TestMaxDepCountEmpty(t *testing.T) {
	got := maxDepCount(nil)
	if got != 0 {
		t.Errorf("maxDepCount(nil) = %d, want 0", got)
	}
}

func TestMaxDepCount(t *testing.T) {
	services := []scanner.Service{
		{
			Name:         "a",
			Integrations: []scanner.Integration{{ClientID: "x"}, {ClientID: "y"}},
			DependedOnBy: []string{"z"},
		},
		{
			Name:         "b",
			Integrations: []scanner.Integration{{ClientID: "x"}},
		},
	}

	got := maxDepCount(services)
	// Service "a" has 2 integrations + 1 dependent = 3
	if got != 3 {
		t.Errorf("maxDepCount() = %d, want 3", got)
	}
}

// ── heatIntensity tests ─────────────────────────────────────────────

func TestHeatIntensityZeroMax(t *testing.T) {
	svc := scanner.Service{Name: "a"}
	got := heatIntensity(svc, 0)
	if got != 0 {
		t.Errorf("heatIntensity with maxDeps=0 = %v, want 0", got)
	}
}

func TestHeatIntensity(t *testing.T) {
	svc := scanner.Service{
		Name:         "a",
		Integrations: []scanner.Integration{{ClientID: "x"}, {ClientID: "y"}},
		DependedOnBy: []string{"z"},
	}

	got := heatIntensity(svc, 6)
	// total = 2 + 1 = 3; intensity = 3/6 = 0.5
	if got != 0.5 {
		t.Errorf("heatIntensity() = %v, want 0.5", got)
	}
}

// ── buildServiceIndex tests ─────────────────────────────────────────

func TestBuildServiceIndex(t *testing.T) {
	services := []scanner.Service{
		{Name: "alpha"},
		{Name: "beta"},
	}

	idx := buildServiceIndex(services)

	if _, ok := idx["alpha"]; !ok {
		t.Error("expected alpha in index")
	}
	if _, ok := idx["beta"]; !ok {
		t.Error("expected beta in index")
	}
	if _, ok := idx["gamma"]; ok {
		t.Error("did not expect gamma in index")
	}
}

func TestBuildServiceIndexEmpty(t *testing.T) {
	idx := buildServiceIndex(nil)
	if len(idx) != 0 {
		t.Errorf("expected empty index, got %d entries", len(idx))
	}
}

// ── buildNodeLabel tests ────────────────────────────────────────────

func TestBuildNodeLabelBasic(t *testing.T) {
	svc := scanner.Service{Name: "my-service"}
	services := []scanner.Service{svc}
	idx := scanner.NewNameIndexFromServices(services)

	got := buildNodeLabel(svc, services, idx)
	if got != "my-service" {
		t.Errorf("buildNodeLabel() = %q, want %q", got, "my-service")
	}
}

func TestBuildNodeLabelWithVersion(t *testing.T) {
	svc := scanner.Service{Name: "my-service", Version: "2.1.0"}
	services := []scanner.Service{svc}
	idx := scanner.NewNameIndexFromServices(services)

	got := buildNodeLabel(svc, services, idx)
	want := "my-service\\n2.1.0"
	if got != want {
		t.Errorf("buildNodeLabel() = %q, want %q", got, want)
	}
}

func TestBuildNodeLabelWithStale(t *testing.T) {
	services := []scanner.Service{
		{
			Name:    "gateway",
			Version: "1.0.0",
			Integrations: []scanner.Integration{
				{ClientID: "backend", SpecVersion: "0.9.0"},
			},
		},
		{Name: "backend", Version: "1.0.0"},
	}
	idx := scanner.NewNameIndexFromServices(services)

	got := buildNodeLabel(services[0], services, idx)
	if !strings.Contains(got, "stale") {
		t.Errorf("buildNodeLabel() = %q, expected stale warning", got)
	}
}

// ── CollectSubgraph tests ───────────────────────────────────────────

func TestCollectSubgraphSingleHop(t *testing.T) {
	services := []scanner.Service{
		{Name: "a", Integrations: []scanner.Integration{{ClientID: "b"}}},
		{Name: "b", Integrations: []scanner.Integration{{ClientID: "c"}}},
		{Name: "c"},
	}
	idx := scanner.NewNameIndexFromServices(services)
	// Compute reverse index manually for the test.
	services[1].DependedOnBy = []string{"a"}
	services[2].DependedOnBy = []string{"b"}

	result := CollectSubgraph("a", services, idx, 1)

	names := make(map[string]bool)
	for _, s := range result {
		names[s.Name] = true
	}
	if !names["a"] {
		t.Error("expected root 'a' in subgraph")
	}
	if !names["b"] {
		t.Error("expected 'b' (1 hop via integration) in subgraph")
	}
	if names["c"] {
		t.Error("did not expect 'c' (2 hops) in subgraph with hops=1")
	}
}

func TestCollectSubgraphTwoHops(t *testing.T) {
	services := []scanner.Service{
		{Name: "a", Integrations: []scanner.Integration{{ClientID: "b"}}},
		{Name: "b", Integrations: []scanner.Integration{{ClientID: "c"}}},
		{Name: "c"},
	}
	idx := scanner.NewNameIndexFromServices(services)
	services[1].DependedOnBy = []string{"a"}
	services[2].DependedOnBy = []string{"b"}

	result := CollectSubgraph("a", services, idx, 2)

	names := make(map[string]bool)
	for _, s := range result {
		names[s.Name] = true
	}
	if !names["a"] || !names["b"] || !names["c"] {
		t.Errorf("expected a, b, c in subgraph with hops=2, got %v", names)
	}
}

func TestCollectSubgraphZeroHops(t *testing.T) {
	services := []scanner.Service{
		{Name: "a", Integrations: []scanner.Integration{{ClientID: "b"}}},
		{Name: "b"},
	}
	idx := scanner.NewNameIndexFromServices(services)

	result := CollectSubgraph("a", services, idx, 0)

	if len(result) != 1 || result[0].Name != "a" {
		t.Errorf("expected only root with hops=0, got %d services", len(result))
	}
}

func TestCollectSubgraphReverseDirection(t *testing.T) {
	services := []scanner.Service{
		{Name: "a"},
		{Name: "b", Integrations: []scanner.Integration{{ClientID: "a"}}},
	}
	idx := scanner.NewNameIndexFromServices(services)
	services[0].DependedOnBy = []string{"b"}

	result := CollectSubgraph("a", services, idx, 1)

	names := make(map[string]bool)
	for _, s := range result {
		names[s.Name] = true
	}
	if !names["b"] {
		t.Error("expected 'b' via reverse (DependedOnBy) in subgraph")
	}
}

func TestCollectSubgraphNonexistentRoot(t *testing.T) {
	services := []scanner.Service{
		{Name: "a"},
		{Name: "b"},
	}
	idx := scanner.NewNameIndexFromServices(services)

	result := CollectSubgraph("nonexistent", services, idx, 3)

	if len(result) != 0 {
		t.Errorf("expected 0 services for nonexistent root, got %d", len(result))
	}
}

// ── WriteDOT tests ──────────────────────────────────────────────────

func TestWriteDOTEmptyServices(t *testing.T) {
	var buf bytes.Buffer
	idx := scanner.NewNameIndexFromServices(nil)

	WriteDOT(&buf, nil, idx, "")

	out := buf.String()
	if !strings.HasPrefix(out, "digraph services {") {
		t.Error("expected DOT output to start with digraph header")
	}
	if !strings.HasSuffix(strings.TrimSpace(out), "}") {
		t.Error("expected DOT output to end with closing brace")
	}
}

func TestWriteDOTWithServices(t *testing.T) {
	services := []scanner.Service{
		{Name: "alpha", Version: "1.0.0", Integrations: []scanner.Integration{{ClientID: "beta"}}},
		{Name: "beta", Version: "2.0.0"},
	}
	idx := scanner.NewNameIndexFromServices(services)

	var buf bytes.Buffer
	WriteDOT(&buf, services, idx, "")

	out := buf.String()
	if !strings.Contains(out, `"alpha"`) {
		t.Error("expected DOT output to contain alpha node")
	}
	if !strings.Contains(out, `"beta"`) {
		t.Error("expected DOT output to contain beta node")
	}
	if !strings.Contains(out, `"alpha" -> "beta"`) {
		t.Error("expected DOT output to contain alpha -> beta edge")
	}
}

func TestWriteDOTHighlight(t *testing.T) {
	services := []scanner.Service{
		{Name: "highlighted"},
	}
	idx := scanner.NewNameIndexFromServices(services)

	var buf bytes.Buffer
	WriteDOT(&buf, services, idx, "highlighted")

	out := buf.String()
	if !strings.Contains(out, "#ff9e64") {
		t.Error("expected highlighted node to have orange color #ff9e64")
	}
}

func TestWriteDOTExternalDependency(t *testing.T) {
	services := []scanner.Service{
		{Name: "alpha", Integrations: []scanner.Integration{{ClientID: "unknown-external"}}},
	}
	idx := scanner.NewNameIndexFromServices(services)

	var buf bytes.Buffer
	WriteDOT(&buf, services, idx, "")

	out := buf.String()
	if !strings.Contains(out, "dashed") {
		t.Error("expected external dependency to be dashed")
	}
	if !strings.Contains(out, "cluster_external") {
		t.Error("expected external subgraph cluster")
	}
}

// ── ExportPNG tests (mocked) ────────────────────────────────────────

func TestExportPNGNoGraphviz(t *testing.T) {
	origLookPath := lookPath
	t.Cleanup(func() { lookPath = origLookPath })

	lookPath = func(file string) (string, error) {
		return "", fmt.Errorf("not found")
	}

	services := []scanner.Service{{Name: "a"}}
	idx := scanner.NewNameIndexFromServices(services)

	_, err := ExportPNG("a", services, idx, 1)
	if err == nil {
		t.Fatal("expected error when graphviz not installed")
	}
	if !strings.Contains(err.Error(), "graphviz") {
		t.Errorf("expected graphviz error message, got: %v", err)
	}
}

func TestExportPNGSuccess(t *testing.T) {
	origLookPath := lookPath
	origRunDot := runDot
	origOpenFile := openFile
	t.Cleanup(func() {
		lookPath = origLookPath
		runDot = origRunDot
		openFile = origOpenFile
	})

	lookPath = func(file string) (string, error) {
		return "/usr/local/bin/dot", nil
	}

	var dotInput []byte
	runDot = func(input []byte, outPath string) error {
		dotInput = input
		// Write a dummy PNG to the output path.
		return os.WriteFile(outPath, []byte("fake-png"), 0644)
	}

	openedPath := ""
	openFile = func(path string) error {
		openedPath = path
		return nil
	}

	services := []scanner.Service{
		{Name: "root", Integrations: []scanner.Integration{{ClientID: "dep"}}},
		{Name: "dep"},
	}
	idx := scanner.NewNameIndexFromServices(services)

	path, err := ExportPNG("root", services, idx, 1)
	if err != nil {
		t.Fatalf("ExportPNG() error: %v", err)
	}
	t.Cleanup(func() { os.Remove(path) })

	if path == "" {
		t.Error("expected non-empty file path")
	}
	if !strings.Contains(string(dotInput), "digraph") {
		t.Error("expected DOT input to contain digraph header")
	}
	if openedPath != path {
		t.Errorf("openFile called with %q, want %q", openedPath, path)
	}
}

func TestExportPNGDotFailure(t *testing.T) {
	origLookPath := lookPath
	origRunDot := runDot
	origOpenFile := openFile
	t.Cleanup(func() {
		lookPath = origLookPath
		runDot = origRunDot
		openFile = origOpenFile
	})

	lookPath = func(file string) (string, error) {
		return "/usr/local/bin/dot", nil
	}

	runDot = func(input []byte, outPath string) error {
		return fmt.Errorf("rendering failed")
	}

	openFile = func(path string) error {
		t.Error("openFile should not be called when dot fails")
		return nil
	}

	services := []scanner.Service{{Name: "a"}}
	idx := scanner.NewNameIndexFromServices(services)

	_, err := ExportPNG("a", services, idx, 1)
	if err == nil {
		t.Fatal("expected error when dot fails")
	}
	if !strings.Contains(err.Error(), "rendering failed") {
		t.Errorf("expected dot error message, got: %v", err)
	}
}

// ── dotError.Error() test ───────────────────────────────────────────

func TestDotErrorMethod(t *testing.T) {
	e := &dotError{msg: "something went wrong"}
	got := e.Error()
	if got != "something went wrong" {
		t.Errorf("dotError.Error() = %q, want %q", got, "something went wrong")
	}
}

func TestDotErrorEmptyMsg(t *testing.T) {
	e := &dotError{msg: ""}
	got := e.Error()
	if got != "" {
		t.Errorf("dotError.Error() = %q, want empty", got)
	}
}

// ── writeDOTNodes with stale service ────────────────────────────────

func TestWriteDOTNodesStaleService(t *testing.T) {
	services := []scanner.Service{
		{
			Name:    "gateway",
			Version: "1.0.0",
			Integrations: []scanner.Integration{
				{ClientID: "backend", SpecVersion: "0.9.0"},
			},
		},
		{Name: "backend", Version: "1.0.0"},
	}
	idx := scanner.NewNameIndexFromServices(services)

	var buf bytes.Buffer
	WriteDOT(&buf, services, idx, "")

	out := buf.String()
	// Stale node should have different fill color
	if !strings.Contains(out, "#2d1f2f") {
		t.Error("expected stale fill color #2d1f2f in DOT output")
	}
	// Should contain stale label
	if !strings.Contains(out, "stale") {
		t.Error("expected stale label in DOT output")
	}
}

// ── ExportPNG with CreateTemp failure is hard to test, but we can test with empty root ──

func TestExportPNGEmptyRoot(t *testing.T) {
	origLookPath := lookPath
	origRunDot := runDot
	origOpenFile := openFile
	t.Cleanup(func() {
		lookPath = origLookPath
		runDot = origRunDot
		openFile = origOpenFile
	})

	lookPath = func(file string) (string, error) {
		return "/usr/local/bin/dot", nil
	}
	runDot = func(input []byte, outPath string) error {
		return os.WriteFile(outPath, []byte("fake-png"), 0644)
	}
	openFile = func(path string) error {
		return nil
	}

	services := []scanner.Service{{Name: "a"}}
	idx := scanner.NewNameIndexFromServices(services)

	// Use "nonexistent" root; should still produce valid output (empty subgraph)
	path, err := ExportPNG("nonexistent", services, idx, 1)
	if path != "" {
		t.Cleanup(func() { os.Remove(path) })
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExportPNGOpenFailure(t *testing.T) {
	origLookPath := lookPath
	origRunDot := runDot
	origOpenFile := openFile
	t.Cleanup(func() {
		lookPath = origLookPath
		runDot = origRunDot
		openFile = origOpenFile
	})

	lookPath = func(file string) (string, error) {
		return "/usr/local/bin/dot", nil
	}

	runDot = func(input []byte, outPath string) error {
		return os.WriteFile(outPath, []byte("fake-png"), 0644)
	}

	openFile = func(path string) error {
		return fmt.Errorf("cannot open")
	}

	services := []scanner.Service{{Name: "a"}}
	idx := scanner.NewNameIndexFromServices(services)

	path, err := ExportPNG("a", services, idx, 1)
	t.Cleanup(func() { os.Remove(path) })

	// ExportPNG returns the path even if open fails.
	if path == "" {
		t.Error("expected file path even when open fails")
	}
	if err == nil {
		t.Fatal("expected error when open fails")
	}
	if !strings.Contains(err.Error(), "open") {
		t.Errorf("expected open error message, got: %v", err)
	}
}
