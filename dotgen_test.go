package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/CheeziCrew/fondue/scanner"
)

func TestWriteDOTDelegation(t *testing.T) {
	services := []scanner.Service{
		{Name: "svc-a", Version: "1.0.0", Integrations: []scanner.Integration{{ClientID: "svc-b"}}},
		{Name: "svc-b", Version: "2.0.0"},
	}
	idx := scanner.NewNameIndexFromServices(services)

	var buf bytes.Buffer
	WriteDOT(&buf, services, idx, "")

	out := buf.String()

	if !strings.HasPrefix(out, "digraph services {") {
		t.Error("expected DOT output to start with digraph header")
	}
	if !strings.Contains(out, `"svc-a"`) {
		t.Error("expected svc-a node in DOT output")
	}
	if !strings.Contains(out, `"svc-b"`) {
		t.Error("expected svc-b node in DOT output")
	}
	if !strings.Contains(out, `"svc-a" -> "svc-b"`) {
		t.Error("expected svc-a -> svc-b edge in DOT output")
	}
	if !strings.HasSuffix(strings.TrimSpace(out), "}") {
		t.Error("expected DOT output to end with closing brace")
	}
}

func TestWriteDOTDelegationWithHighlight(t *testing.T) {
	services := []scanner.Service{
		{Name: "target"},
	}
	idx := scanner.NewNameIndexFromServices(services)

	var buf bytes.Buffer
	WriteDOT(&buf, services, idx, "target")

	out := buf.String()
	if !strings.Contains(out, "#ff9e64") {
		t.Error("expected highlight color in DOT output when highlight is set")
	}
}

func TestWriteDOTDelegationEmpty(t *testing.T) {
	idx := scanner.NewNameIndexFromServices(nil)

	var buf bytes.Buffer
	WriteDOT(&buf, nil, idx, "")

	out := buf.String()
	if !strings.Contains(out, "digraph") {
		t.Error("expected valid DOT even with no services")
	}
}
