package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

const (
	testCaseData = "case-data"
	testPomXML   = "pom.xml"
)

// ── NameIndex tests ─────────────────────────────────────────────────

func TestNameIndexExactMatch(t *testing.T) {
	idx := NewNameIndex(map[string]bool{"party": true, testCaseData: true})

	tests := []struct {
		input string
		want  string
	}{
		{"party", "party"},
		{testCaseData, testCaseData},
		{"casedata", testCaseData},      // hyphen stripped
		{"case.data", testCaseData},     // dot variant
		{"Party", "party"},             // case insensitive
		{"CASEDATA", testCaseData},      // uppercase + stripped
		{"nonexistent", ""},            // no match
	}

	for _, tt := range tests {
		got := idx.Resolve(tt.input)
		if got != tt.want {
			t.Errorf("Resolve(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNameIndexPluralSingular(t *testing.T) {
	idx := NewNameIndex(map[string]bool{
		"relations": true,
		"party":     true,
	})

	tests := []struct {
		input string
		want  string
	}{
		{"relations", "relations"},
		{"relation", "relations"},    // singular -> plural
		{"partys", "party"},          // plural -> singular
		{"party", "party"},
	}

	for _, tt := range tests {
		got := idx.Resolve(tt.input)
		if got != tt.want {
			t.Errorf("Resolve(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNameIndexIsInternal(t *testing.T) {
	idx := NewNameIndex(map[string]bool{"party": true})

	if !idx.IsInternal("party") {
		t.Error("expected party to be internal")
	}
	if idx.IsInternal("unknown-service") {
		t.Error("expected unknown-service to not be internal")
	}
}

// ── extractPackageName tests ────────────────────────────────────────

func TestExtractPackageName(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"se/sundsvall/foo/integration/casedata/configuration/Config.java", "casedata"},
		{"se/sundsvall/foo/integration/party/PartyClient.java", "party"},
		{"se/sundsvall/foo/service/SomeService.java", ""},
		{"integration/direct/File.java", "direct"},
	}

	for _, tt := range tests {
		got := extractPackageName(tt.path)
		if got != tt.want {
			t.Errorf("extractPackageName(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

// ── normalizeSpecFilename tests ─────────────────────────────────────

func TestNormalizeSpecFilename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"party-api-1.3.0.yaml", "party"},
		{"case-data-api.yaml", "casedata"},
		{"messaging.yml", "messaging"},
		{"api-access-mapper.yaml", "accessmapper"},
		{"notes-2.0.yaml", "notes"},
	}

	for _, tt := range tests {
		got := normalizeSpecFilename(tt.input)
		if got != tt.want {
			t.Errorf("normalizeSpecFilename(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// ── extractPomVersion tests ─────────────────────────────────────────

func TestExtractPomVersion(t *testing.T) {
	tmpDir := t.TempDir()

	// Standard pom with parent version + project version
	pom := `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
    <parent>
        <version>3.0.0</version>
    </parent>
    <version>2.1.0</version>
</project>`

	os.WriteFile(filepath.Join(tmpDir, testPomXML), []byte(pom), 0644)

	got := extractPomVersion(tmpDir)
	if got != "2.1.0" {
		t.Errorf("extractPomVersion() = %q, want %q", got, "2.1.0")
	}
}

func TestExtractPomVersionPlaceholderSkipped(t *testing.T) {
	tmpDir := t.TempDir()

	pom := `<?xml version="1.0" encoding="UTF-8"?>
<project>
    <version>@project.version@</version>
</project>`

	os.WriteFile(filepath.Join(tmpDir, testPomXML), []byte(pom), 0644)

	got := extractPomVersion(tmpDir)
	if got != "" {
		t.Errorf("extractPomVersion() = %q, want empty", got)
	}
}

// ── extractSpecVersion tests ────────────────────────────────────────

func TestExtractSpecVersion(t *testing.T) {
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "api.yaml")

	spec := `openapi: "3.0.0"
info:
  title: Party API
  version: 1.3.0
paths:`

	os.WriteFile(specFile, []byte(spec), 0644)

	got := extractSpecVersion(specFile)
	if got != "1.3.0" {
		t.Errorf("extractSpecVersion() = %q, want %q", got, "1.3.0")
	}
}

func TestExtractSpecVersionQuoted(t *testing.T) {
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "api.yaml")

	spec := `openapi: "3.0.0"
info:
  title: Test
  version: "2.0.1"
paths:`

	os.WriteFile(specFile, []byte(spec), 0644)

	got := extractSpecVersion(specFile)
	if got != "2.0.1" {
		t.Errorf("extractSpecVersion() = %q, want %q", got, "2.0.1")
	}
}

// ── StaleCount tests ────────────────────────────────────────────────

func TestStaleCount(t *testing.T) {
	services := []Service{
		{
			Name:    "api-gateway",
			Version: "1.0.0",
			Integrations: []Integration{
				{ClientID: "party", SpecVersion: "1.3.0"},
				{ClientID: "messaging", SpecVersion: "2.0.0"},
			},
		},
		{Name: "party", Version: "1.3.0"},
		{Name: "messaging", Version: "2.1.0"}, // stale: 2.0.0 != 2.1.0
	}

	idx := NewNameIndexFromServices(services)
	count := StaleCount(&services[0], services, idx)

	if count != 1 {
		t.Errorf("StaleCount() = %d, want 1", count)
	}
}

// ── Full scan integration test ──────────────────────────────────────

func TestScanEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := ScanConfig{
		RepoPrefixes:    []string{"api-service-"},
		StandaloneRepos: map[string]string{},
	}

	services, idx, err := Scan(tmpDir, cfg)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if len(services) != 0 {
		t.Errorf("expected 0 services, got %d", len(services))
	}
	if idx == nil {
		t.Error("expected non-nil NameIndex")
	}
}

func TestScanWithService(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a minimal service structure
	svcDir := filepath.Join(tmpDir, "api-service-test-svc")
	javaDir := filepath.Join(svcDir, "src", "main", "java", "se", "sundsvall", "test", "integration", "party")
	os.MkdirAll(javaDir, 0755)

	// Write a Java file with CLIENT_ID
	javaFile := `package se.sundsvall.test.integration.party;
public class Config {
    public static final String CLIENT_ID = "party";
}`
	os.WriteFile(filepath.Join(javaDir, "Config.java"), []byte(javaFile), 0644)

	// Write a pom.xml
	pom := `<?xml version="1.0" encoding="UTF-8"?>
<project>
    <version>1.0.0</version>
</project>`
	os.WriteFile(filepath.Join(svcDir, testPomXML), []byte(pom), 0644)

	cfg := ScanConfig{
		RepoPrefixes:    []string{"api-service-"},
		StandaloneRepos: map[string]string{},
	}

	services, idx, err := Scan(tmpDir, cfg)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if len(services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(services))
	}

	svc := services[0]
	if svc.Name != "test-svc" {
		t.Errorf("Name = %q, want %q", svc.Name, "test-svc")
	}
	if svc.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", svc.Version, "1.0.0")
	}
	if len(svc.Integrations) != 1 {
		t.Fatalf("expected 1 integration, got %d", len(svc.Integrations))
	}
	if svc.Integrations[0].ClientID != "party" {
		t.Errorf("ClientID = %q, want %q", svc.Integrations[0].ClientID, "party")
	}
	if idx == nil {
		t.Error("expected non-nil NameIndex")
	}
}
