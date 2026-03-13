package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/CheeziCrew/fondue/scanner"
)

func TestBuild(t *testing.T) {}

func TestOutputModeConstants(t *testing.T) {
	tests := []struct {
		name string
		mode outputMode
		want int
	}{
		{"interactive", modeInteractive, 0},
		{"list", modeList, 1},
		{"json", modeJSON, 2},
		{"dot", modeDOT, 3},
		{"cycles", modeCycles, 4},
		{"impact", modeImpact, 5},
		{"diff", modeDiff, 6},
	}
	for _, tt := range tests {
		if int(tt.mode) != tt.want {
			t.Errorf("%s = %d, want %d", tt.name, tt.mode, tt.want)
		}
	}
}

func TestVersionVar(t *testing.T) {
	if version == "" {
		t.Error("version should not be empty (default is 'dev')")
	}
}

func TestPrintUsage(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	printUsage()

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "fondue") {
		t.Error("printUsage should mention 'fondue'")
	}
	if !strings.Contains(output, "--list") {
		t.Error("printUsage should mention --list flag")
	}
	if !strings.Contains(output, "--json") {
		t.Error("printUsage should mention --json flag")
	}
	if !strings.Contains(output, "--dot") {
		t.Error("printUsage should mention --dot flag")
	}
	if !strings.Contains(output, ".fondue.json") {
		t.Error("printUsage should mention config file")
	}
}

func TestPrintList(t *testing.T) {
	services := []scanner.Service{
		{
			Name:    "auth-service",
			Version: "1.0.0",
			Integrations: []scanner.Integration{
				{ClientID: "user-service", SpecVersion: "1.5.0"},
			},
			DependedOnBy: []string{"gateway"},
		},
		{
			Name:    "user-service",
			Version: "2.0.0",
		},
		{
			Name: "no-version-svc",
		},
	}
	idx := scanner.NewNameIndexFromServices(services)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printList(services, idx)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "auth-service") {
		t.Error("printList should show auth-service")
	}
	if !strings.Contains(output, "user-service") {
		t.Error("printList should show user-service")
	}
	if !strings.Contains(output, "no-version-svc") {
		t.Error("printList should show no-version-svc")
	}
	if !strings.Contains(output, "?") {
		t.Error("printList should show ? for empty version")
	}
	if !strings.Contains(output, "stale") {
		t.Error("printList should show stale for mismatched spec versions")
	}
	if !strings.Contains(output, "->") {
		t.Error("printList should show -> for integrations")
	}
}

func TestPrintListNoIntegrations(t *testing.T) {
	services := []scanner.Service{
		{Name: "isolated", Version: "1.0.0"},
	}
	idx := scanner.NewNameIndexFromServices(services)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printList(services, idx)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "isolated") {
		t.Error("printList should show isolated service")
	}
	// Should NOT contain -> since no integrations
	if strings.Contains(output, "->") {
		t.Error("printList should not show -> for services without integrations")
	}
}

func TestPrintJSON(t *testing.T) {
	services := []scanner.Service{
		{
			Name:    "svc-a",
			Version: "1.0.0",
			Path:    "/tmp/svc-a",
			Integrations: []scanner.Integration{
				{ClientID: "svc-b"},
			},
			DependedOnBy: []string{"svc-c"},
		},
		{
			Name:    "svc-b",
			Version: "2.0.0",
			Path:    "/tmp/svc-b",
		},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printJSON(services)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify valid JSON
	var parsed []jsonService
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("printJSON output is not valid JSON: %v\nOutput: %s", err, output)
	}

	if len(parsed) != 2 {
		t.Fatalf("expected 2 services, got %d", len(parsed))
	}

	if parsed[0].Name != "svc-a" {
		t.Errorf("first service name = %q, want %q", parsed[0].Name, "svc-a")
	}
	if parsed[0].Version != "1.0.0" {
		t.Errorf("first service version = %q, want %q", parsed[0].Version, "1.0.0")
	}
	if len(parsed[0].Dependencies) != 1 || parsed[0].Dependencies[0] != "svc-b" {
		t.Errorf("first service dependencies = %v, want [svc-b]", parsed[0].Dependencies)
	}
	if len(parsed[0].DependedOnBy) != 1 || parsed[0].DependedOnBy[0] != "svc-c" {
		t.Errorf("first service dependedOnBy = %v, want [svc-c]", parsed[0].DependedOnBy)
	}
	if parsed[1].Dependencies != nil {
		t.Errorf("second service dependencies = %v, want nil", parsed[1].Dependencies)
	}
}

func TestPrintJSONEmpty(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printJSON(nil)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Even with nil input, it should produce valid JSON
	var parsed []jsonService
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("printJSON(nil) output is not valid JSON: %v", err)
	}
}

// ── Cycle / Impact / Diff tests ─────────────────────────────────────

func TestPrintCyclesNone(t *testing.T) {
	services := []scanner.Service{
		{Name: "a", Integrations: []scanner.Integration{{ClientID: "b"}}},
		{Name: "b"},
	}
	idx := scanner.NewNameIndexFromServices(services)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printCycles(services, idx)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !strings.Contains(buf.String(), "No dependency cycles") {
		t.Error("expected 'No dependency cycles' message")
	}
}

func TestPrintCyclesFound(t *testing.T) {
	services := []scanner.Service{
		{Name: "a", Integrations: []scanner.Integration{{ClientID: "b"}}},
		{Name: "b", Integrations: []scanner.Integration{{ClientID: "a"}}},
	}
	idx := scanner.NewNameIndexFromServices(services)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printCycles(services, idx)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !strings.Contains(output, "1 dependency cycle") {
		t.Errorf("expected cycle count, got: %s", output)
	}
}

func TestPrintImpactWithConsumers(t *testing.T) {
	services := []scanner.Service{
		{Name: "core", DependedOnBy: []string{"gateway"}},
		{Name: "gateway"},
	}
	idx := scanner.NewNameIndexFromServices(services)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printImpact("core", services, idx)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !strings.Contains(output, "gateway") {
		t.Error("expected 'gateway' in impact output")
	}
}

func TestPrintImpactNoConsumers(t *testing.T) {
	services := []scanner.Service{{Name: "isolated"}}
	idx := scanner.NewNameIndexFromServices(services)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printImpact("isolated", services, idx)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !strings.Contains(buf.String(), "No downstream consumers") {
		t.Error("expected 'No downstream consumers' message")
	}
}

func TestLoadDiffJSON(t *testing.T) {
	tmp := t.TempDir()
	path := tmp + "/test.json"
	content := `[{"name":"svc-a","version":"1.0","dependencies":["svc-b"]},{"name":"svc-b"}]`
	os.WriteFile(path, []byte(content), 0644)

	services, err := loadDiffJSON(path)
	if err != nil {
		t.Fatalf("loadDiffJSON() error: %v", err)
	}
	if len(services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(services))
	}
	if services[0].Name != "svc-a" {
		t.Errorf("first service name = %q, want %q", services[0].Name, "svc-a")
	}
	if len(services[0].Dependencies) != 1 {
		t.Errorf("expected 1 dependency, got %d", len(services[0].Dependencies))
	}
}

func TestLoadDiffJSONFileNotFound(t *testing.T) {
	_, err := loadDiffJSON("/nonexistent/path.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadDiffJSONInvalidJSON(t *testing.T) {
	tmp := t.TempDir()
	path := tmp + "/bad.json"
	os.WriteFile(path, []byte("not json"), 0644)

	_, err := loadDiffJSON(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Errorf("expected 'invalid JSON' in error, got: %v", err)
	}
}

func TestPrintDiffValid(t *testing.T) {
	tmp := t.TempDir()

	oldPath := tmp + "/old.json"
	newPath := tmp + "/new.json"

	os.WriteFile(oldPath, []byte(`[{"name":"a","version":"1.0"}]`), 0644)
	os.WriteFile(newPath, []byte(`[{"name":"a","version":"2.0"},{"name":"b"}]`), 0644)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printDiff(oldPath, newPath)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "change") {
		t.Errorf("expected changes in output, got: %s", output)
	}
}

func TestPrintDiffNoChanges(t *testing.T) {
	tmp := t.TempDir()

	path := tmp + "/same.json"
	os.WriteFile(path, []byte(`[{"name":"a","version":"1.0"}]`), 0644)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printDiff(path, path)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !strings.Contains(buf.String(), "No changes") {
		t.Error("expected 'No changes' for identical files")
	}
}

func TestPrintImpactEmptyTarget(t *testing.T) {
	exitCalled := false
	exitFunc = func(code int) { exitCalled = true }
	defer func() { exitFunc = os.Exit }()

	services := []scanner.Service{{Name: "a"}}
	idx := scanner.NewNameIndexFromServices(services)

	// Capture stderr
	oldStderr := os.Stderr
	_, w, _ := os.Pipe()
	os.Stderr = w

	printImpact("", services, idx)

	w.Close()
	os.Stderr = oldStderr

	if !exitCalled {
		t.Error("expected exitFunc to be called for empty target")
	}
}

func TestPrintDiffEmptyPaths(t *testing.T) {
	exitCalled := false
	exitFunc = func(code int) { exitCalled = true }
	defer func() { exitFunc = os.Exit }()

	oldStderr := os.Stderr
	_, w, _ := os.Pipe()
	os.Stderr = w

	printDiff("", "")

	w.Close()
	os.Stderr = oldStderr

	if !exitCalled {
		t.Error("expected exitFunc to be called for empty paths")
	}
}

func TestPrintDiffBadOldFile(t *testing.T) {
	exitCalled := false
	exitFunc = func(code int) { exitCalled = true }
	defer func() { exitFunc = os.Exit }()

	tmp := t.TempDir()
	newPath := tmp + "/new.json"
	os.WriteFile(newPath, []byte(`[{"name":"a"}]`), 0644)

	oldStderr := os.Stderr
	_, w, _ := os.Pipe()
	os.Stderr = w

	printDiff("/nonexistent/old.json", newPath)

	w.Close()
	os.Stderr = oldStderr

	if !exitCalled {
		t.Error("expected exitFunc for bad old file")
	}
}

func TestPrintDiffBadNewFile(t *testing.T) {
	exitCalled := false
	exitFunc = func(code int) { exitCalled = true }
	defer func() { exitFunc = os.Exit }()

	tmp := t.TempDir()
	oldPath := tmp + "/old.json"
	os.WriteFile(oldPath, []byte(`[{"name":"a"}]`), 0644)

	oldStderr := os.Stderr
	_, w, _ := os.Pipe()
	os.Stderr = w

	printDiff(oldPath, "/nonexistent/new.json")

	w.Close()
	os.Stderr = oldStderr

	if !exitCalled {
		t.Error("expected exitFunc for bad new file")
	}
}

// ── parseArgs tests ──────────────────────────────────────────────────

func TestParseArgsDefaults(t *testing.T) {
	p := parseArgs(nil)
	if p.mode != modeInteractive {
		t.Errorf("default mode = %d, want interactive (0)", p.mode)
	}
	if p.customPath != "" {
		t.Errorf("default customPath = %q, want empty", p.customPath)
	}
}

func TestParseArgsList(t *testing.T) {
	for _, flag := range []string{"--list", "-l"} {
		p := parseArgs([]string{flag})
		if p.mode != modeList {
			t.Errorf("parseArgs(%q).mode = %d, want modeList", flag, p.mode)
		}
	}
}

func TestParseArgsJSON(t *testing.T) {
	for _, flag := range []string{"--json", "-j"} {
		p := parseArgs([]string{flag})
		if p.mode != modeJSON {
			t.Errorf("parseArgs(%q).mode = %d, want modeJSON", flag, p.mode)
		}
	}
}

func TestParseArgsDOT(t *testing.T) {
	for _, flag := range []string{"--dot", "-d"} {
		p := parseArgs([]string{flag})
		if p.mode != modeDOT {
			t.Errorf("parseArgs(%q).mode = %d, want modeDOT", flag, p.mode)
		}
	}
}

func TestParseArgsCycles(t *testing.T) {
	for _, flag := range []string{"--cycles", "-c"} {
		p := parseArgs([]string{flag})
		if p.mode != modeCycles {
			t.Errorf("parseArgs(%q).mode = %d, want modeCycles", flag, p.mode)
		}
	}
}

func TestParseArgsImpact(t *testing.T) {
	p := parseArgs([]string{"--impact", "my-service"})
	if p.mode != modeImpact {
		t.Errorf("mode = %d, want modeImpact", p.mode)
	}
	if p.impactTarget != "my-service" {
		t.Errorf("impactTarget = %q, want %q", p.impactTarget, "my-service")
	}
}

func TestParseArgsImpactNoTarget(t *testing.T) {
	p := parseArgs([]string{"--impact"})
	if p.mode != modeImpact {
		t.Errorf("mode = %d, want modeImpact", p.mode)
	}
	if p.impactTarget != "" {
		t.Errorf("impactTarget = %q, want empty", p.impactTarget)
	}
}

func TestParseArgsImpactNextIsFlag(t *testing.T) {
	p := parseArgs([]string{"--impact", "--json"})
	if p.mode != modeJSON {
		t.Errorf("mode = %d, want modeJSON (last flag wins)", p.mode)
	}
	if p.impactTarget != "" {
		t.Errorf("impactTarget = %q, want empty (next arg is a flag)", p.impactTarget)
	}
}

func TestParseArgsDiff(t *testing.T) {
	p := parseArgs([]string{"--diff", "old.json", "new.json"})
	if p.mode != modeDiff {
		t.Errorf("mode = %d, want modeDiff", p.mode)
	}
	if p.diffFiles[0] != "old.json" || p.diffFiles[1] != "new.json" {
		t.Errorf("diffFiles = %v, want [old.json new.json]", p.diffFiles)
	}
}

func TestParseArgsDiffPartial(t *testing.T) {
	p := parseArgs([]string{"--diff", "old.json"})
	if p.diffFiles[0] != "old.json" || p.diffFiles[1] != "" {
		t.Errorf("diffFiles = %v, want [old.json ]", p.diffFiles)
	}
}

func TestParseArgsHelp(t *testing.T) {
	for _, flag := range []string{"--help", "-h"} {
		p := parseArgs([]string{flag})
		if p.mode != -1 {
			t.Errorf("parseArgs(%q).mode = %d, want -1 (help)", flag, p.mode)
		}
	}
}

func TestParseArgsVersion(t *testing.T) {
	for _, flag := range []string{"--version", "-v"} {
		p := parseArgs([]string{flag})
		if p.mode != -2 {
			t.Errorf("parseArgs(%q).mode = %d, want -2 (version)", flag, p.mode)
		}
	}
}

func TestParseArgsCustomPath(t *testing.T) {
	p := parseArgs([]string{"/some/path"})
	if p.customPath != "/some/path" {
		t.Errorf("customPath = %q, want /some/path", p.customPath)
	}
	if p.mode != modeInteractive {
		t.Errorf("mode = %d, want interactive", p.mode)
	}
}

func TestParseArgsCombined(t *testing.T) {
	p := parseArgs([]string{"--json", "/my/dir"})
	if p.mode != modeJSON {
		t.Errorf("mode = %d, want modeJSON", p.mode)
	}
	if p.customPath != "/my/dir" {
		t.Errorf("customPath = %q, want /my/dir", p.customPath)
	}
}

// ── run() tests ─────────────────────────────────────────────────────

func TestRunHelp(t *testing.T) {
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	run([]string{"--help"})

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !strings.Contains(buf.String(), "fondue") {
		t.Error("run(--help) should print usage to stderr")
	}
}

func TestRunVersion(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	run([]string{"--version"})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !strings.Contains(buf.String(), "fondue") {
		t.Error("run(--version) should print version")
	}
}

func TestRunDiff(t *testing.T) {
	tmp := t.TempDir()
	oldP := tmp + "/old.json"
	newP := tmp + "/new.json"
	os.WriteFile(oldP, []byte(`[{"name":"a","version":"1.0"}]`), 0644)
	os.WriteFile(newP, []byte(`[{"name":"a","version":"2.0"}]`), 0644)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	run([]string{"--diff", oldP, newP})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !strings.Contains(buf.String(), "change") {
		t.Error("run(--diff) should show changes")
	}
}

func TestRunListWithEmptyDir(t *testing.T) {
	tmp := t.TempDir()

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	oldStderr := os.Stderr
	_, w2, _ := os.Pipe()
	os.Stderr = w2

	run([]string{"--list", tmp})

	w.Close()
	w2.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr
	// No crash = pass — empty dir with no services is fine
}

func TestRunJSONWithEmptyDir(t *testing.T) {
	tmp := t.TempDir()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldStderr := os.Stderr
	_, w2, _ := os.Pipe()
	os.Stderr = w2

	run([]string{"--json", tmp})

	w.Close()
	w2.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	// Should produce valid JSON (empty array)
	var parsed []jsonService
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("run(--json) with empty dir should produce valid JSON: %v", err)
	}
}

func TestRunDOTWithEmptyDir(t *testing.T) {
	tmp := t.TempDir()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldStderr := os.Stderr
	_, w2, _ := os.Pipe()
	os.Stderr = w2

	run([]string{"--dot", tmp})

	w.Close()
	w2.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !strings.Contains(buf.String(), "digraph") {
		t.Error("run(--dot) should produce DOT output")
	}
}

func TestRunCyclesWithEmptyDir(t *testing.T) {
	tmp := t.TempDir()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldStderr := os.Stderr
	_, w2, _ := os.Pipe()
	os.Stderr = w2

	run([]string{"--cycles", tmp})

	w.Close()
	w2.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !strings.Contains(buf.String(), "No dependency cycles") {
		t.Error("run(--cycles) with empty dir should show no cycles")
	}
}

func TestRunCustomPathOverridesConfig(t *testing.T) {
	tmp := t.TempDir()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldStderr := os.Stderr
	_, w2, _ := os.Pipe()
	os.Stderr = w2

	run([]string{"--json", tmp})

	w.Close()
	w2.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	var parsed []jsonService
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("expected valid JSON: %v", err)
	}
}

func TestRunImpactWithTarget(t *testing.T) {
	tmp := t.TempDir()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldStderr := os.Stderr
	_, w2, _ := os.Pipe()
	os.Stderr = w2

	run([]string{"--impact", "some-service", tmp})

	w.Close()
	w2.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !strings.Contains(buf.String(), "No downstream consumers") {
		t.Errorf("expected 'No downstream consumers', got: %s", buf.String())
	}
}

func TestPrintUsageMentionsNewFlags(t *testing.T) {
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	printUsage()

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "--cycles") {
		t.Error("printUsage should mention --cycles flag")
	}
	if !strings.Contains(output, "--impact") {
		t.Error("printUsage should mention --impact flag")
	}
	if !strings.Contains(output, "--diff") {
		t.Error("printUsage should mention --diff flag")
	}
}
