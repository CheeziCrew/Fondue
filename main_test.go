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
