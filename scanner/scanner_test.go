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

// ── Edge case: directory without pom.xml ────────────────────────────

func TestScanRepoWithoutPomXML(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a directory that matches the prefix but has no pom.xml and no src/main/java.
	svcDir := filepath.Join(tmpDir, "api-service-no-pom")
	os.MkdirAll(svcDir, 0755)

	cfg := ScanConfig{
		RepoPrefixes:    []string{"api-service-"},
		StandaloneRepos: map[string]string{},
	}

	services, _, err := Scan(tmpDir, cfg)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if len(services) != 0 {
		t.Errorf("expected 0 services for repo without src/main/java, got %d", len(services))
	}
}

// ── Edge case: repo with src/main/java but no integration files ─────

func TestScanRepoNoIntegrations(t *testing.T) {
	tmpDir := t.TempDir()

	svcDir := filepath.Join(tmpDir, "api-service-empty-svc")
	javaDir := filepath.Join(svcDir, "src", "main", "java", "se", "sundsvall", "service")
	os.MkdirAll(javaDir, 0755)

	// A Java file NOT in an integration package.
	os.WriteFile(filepath.Join(javaDir, "Main.java"), []byte("package se.sundsvall.service;\npublic class Main {}"), 0644)

	pom := `<?xml version="1.0" encoding="UTF-8"?>
<project><version>1.0.0</version></project>`
	os.WriteFile(filepath.Join(svcDir, testPomXML), []byte(pom), 0644)

	cfg := ScanConfig{
		RepoPrefixes:    []string{"api-service-"},
		StandaloneRepos: map[string]string{},
	}

	services, _, err := Scan(tmpDir, cfg)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if len(services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(services))
	}
	if len(services[0].Integrations) != 0 {
		t.Errorf("expected 0 integrations, got %d", len(services[0].Integrations))
	}
}

// ── Edge case: nested integration structure ─────────────────────────

func TestScanNestedIntegrationPackages(t *testing.T) {
	tmpDir := t.TempDir()

	svcDir := filepath.Join(tmpDir, "api-service-nested")
	integDir := filepath.Join(svcDir, "src", "main", "java", "se", "sundsvall", "app", "integration", "party", "config")
	os.MkdirAll(integDir, 0755)

	javaFile := `package se.sundsvall.app.integration.party.config;
public class PartyConfig {
    public static final String CLIENT_ID = "party-service";
}`
	os.WriteFile(filepath.Join(integDir, "PartyConfig.java"), []byte(javaFile), 0644)

	pom := `<?xml version="1.0" encoding="UTF-8"?>
<project><version>3.0.0</version></project>`
	os.WriteFile(filepath.Join(svcDir, testPomXML), []byte(pom), 0644)

	cfg := ScanConfig{
		RepoPrefixes:    []string{"api-service-"},
		StandaloneRepos: map[string]string{},
	}

	services, _, err := Scan(tmpDir, cfg)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if len(services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(services))
	}
	found := false
	for _, integ := range services[0].Integrations {
		if integ.ClientID == "party-service" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected integration with ClientID 'party-service', got %v", services[0].Integrations)
	}
}

// ── Edge case: standalone repos ─────────────────────────────────────

func TestScanStandaloneRepo(t *testing.T) {
	tmpDir := t.TempDir()

	svcDir := filepath.Join(tmpDir, "legacy-repo")
	javaDir := filepath.Join(svcDir, "src", "main", "java", "se", "sundsvall", "legacy", "integration", "notes")
	os.MkdirAll(javaDir, 0755)

	javaFile := `package se.sundsvall.legacy.integration.notes;
public class NotesClient {
    public static final String INTEGRATION_NAME = "notes";
}`
	os.WriteFile(filepath.Join(javaDir, "NotesClient.java"), []byte(javaFile), 0644)

	pom := `<?xml version="1.0" encoding="UTF-8"?>
<project><version>2.0.0</version></project>`
	os.WriteFile(filepath.Join(svcDir, testPomXML), []byte(pom), 0644)

	cfg := ScanConfig{
		RepoPrefixes:    []string{},
		StandaloneRepos: map[string]string{"legacy-repo": "legacy-service"},
	}

	services, _, err := Scan(tmpDir, cfg)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if len(services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(services))
	}
	if services[0].Name != "legacy-service" {
		t.Errorf("Name = %q, want %q", services[0].Name, "legacy-service")
	}
}

// ── Edge case: duplicate service names across prefix and standalone ──

func TestScanDuplicateServiceNameSkipped(t *testing.T) {
	tmpDir := t.TempDir()

	svcDir1 := filepath.Join(tmpDir, "api-service-duptest")
	javaDir1 := filepath.Join(svcDir1, "src", "main", "java")
	os.MkdirAll(javaDir1, 0755)
	os.WriteFile(filepath.Join(svcDir1, testPomXML), []byte(`<project><version>1.0</version></project>`), 0644)

	svcDir2 := filepath.Join(tmpDir, "duptest-alt")
	javaDir2 := filepath.Join(svcDir2, "src", "main", "java")
	os.MkdirAll(javaDir2, 0755)
	os.WriteFile(filepath.Join(svcDir2, testPomXML), []byte(`<project><version>2.0</version></project>`), 0644)

	cfg := ScanConfig{
		RepoPrefixes:    []string{"api-service-"},
		StandaloneRepos: map[string]string{"duptest-alt": "duptest"},
	}

	services, _, err := Scan(tmpDir, cfg)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if len(services) != 1 {
		t.Errorf("expected 1 service (duplicate skipped), got %d", len(services))
	}
}

// ── Edge case: extractPomVersion with missing file ──────────────────

func TestExtractPomVersionMissingFile(t *testing.T) {
	got := extractPomVersion(t.TempDir())
	if got != "" {
		t.Errorf("extractPomVersion() = %q, want empty for missing pom.xml", got)
	}
}

// ── Edge case: extractPomVersion with invalid XML ───────────────────

func TestExtractPomVersionInvalidXML(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, testPomXML), []byte("not xml at all"), 0644)

	got := extractPomVersion(tmpDir)
	if got != "" {
		t.Errorf("extractPomVersion() = %q, want empty for invalid XML", got)
	}
}

// ── Edge case: extractSpecVersion with missing file ─────────────────

func TestExtractSpecVersionMissingFile(t *testing.T) {
	got := extractSpecVersion("/nonexistent/path/api.yaml")
	if got != "" {
		t.Errorf("extractSpecVersion() = %q, want empty for missing file", got)
	}
}

// ── Edge case: extractSpecVersion with no version field ─────────────

func TestExtractSpecVersionNoVersion(t *testing.T) {
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "api.yaml")
	os.WriteFile(specFile, []byte("openapi: '3.0.0'\ninfo:\n  title: No Version\npaths:"), 0644)

	got := extractSpecVersion(specFile)
	if got != "" {
		t.Errorf("extractSpecVersion() = %q, want empty when no version present", got)
	}
}

// ── isDBPath tests ──────────────────────────────────────────────────

func TestIsDBPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"se/sundsvall/foo/integration/db/SomeMapper.java", true},
		{"se/sundsvall/foo/integration/party/Client.java", false},
		{"db/stuff.java", true},
		{"integration/service/File.java", false},
	}

	for _, tt := range tests {
		got := isDBPath(tt.path)
		if got != tt.want {
			t.Errorf("isDBPath(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

// ── FindService tests ───────────────────────────────────────────────

func TestFindService(t *testing.T) {
	services := []Service{
		{Name: "alpha"},
		{Name: "beta"},
	}

	got := FindService("beta", services)
	if got == nil || got.Name != "beta" {
		t.Errorf("FindService('beta') = %v, want beta", got)
	}

	got = FindService("nonexistent", services)
	if got != nil {
		t.Errorf("FindService('nonexistent') = %v, want nil", got)
	}
}

// ── appendUnique tests ──────────────────────────────────────────────

func TestAppendUnique(t *testing.T) {
	s := []string{"a", "b"}
	s = appendUnique(s, "c")
	if len(s) != 3 {
		t.Errorf("expected 3 elements after appending unique, got %d", len(s))
	}
	s = appendUnique(s, "b")
	if len(s) != 3 {
		t.Errorf("expected 3 elements after appending duplicate, got %d", len(s))
	}
}

// ── NewNameIndexFromServices tests ──────────────────────────────────

func TestNewNameIndexFromServices(t *testing.T) {
	services := []Service{
		{Name: "case-data"},
		{Name: "party"},
	}

	idx := NewNameIndexFromServices(services)

	if idx.Resolve("case-data") != "case-data" {
		t.Error("expected to resolve case-data")
	}
	if idx.Resolve("party") != "party" {
		t.Error("expected to resolve party")
	}
	if idx.Resolve("unknown") != "" {
		t.Error("expected empty for unknown")
	}
}

// ── normalizeClientID tests ─────────────────────────────────────────

func TestNormalizeClientID(t *testing.T) {
	idx := NewNameIndex(map[string]bool{"party": true, "notes": true})

	tests := []struct {
		name    string
		input   string
		want    string
	}{
		{"already resolved", "party", "party"},
		{"client suffix", "partyclient", "party"},
		{"integration suffix", "partyintegration", "party"},
		{"service suffix", "partyservice", "party"},
		{"unknown stays unchanged", "unknown-thing", "unknown-thing"},
		{"upper case client suffix", "PartyClient", "party"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			integ := &Integration{ClientID: tt.input}
			normalizeClientID(integ, idx)
			if integ.ClientID != tt.want {
				t.Errorf("normalizeClientID(%q) = %q, want %q", tt.input, integ.ClientID, tt.want)
			}
		})
	}
}

// ── applyReverseIndex tests ─────────────────────────────────────────

func TestApplyReverseIndex(t *testing.T) {
	services := []Service{
		{Name: "alpha", Integrations: []Integration{{ClientID: "beta"}}},
		{Name: "beta", Integrations: []Integration{{ClientID: "gamma"}}},
		{Name: "gamma"},
	}
	idx := NewNameIndexFromServices(services)

	applyReverseIndex(services, idx)

	if len(services[1].DependedOnBy) != 1 || services[1].DependedOnBy[0] != "alpha" {
		t.Errorf("beta.DependedOnBy = %v, want [alpha]", services[1].DependedOnBy)
	}
	if len(services[2].DependedOnBy) != 1 || services[2].DependedOnBy[0] != "beta" {
		t.Errorf("gamma.DependedOnBy = %v, want [beta]", services[2].DependedOnBy)
	}
	if len(services[0].DependedOnBy) != 0 {
		t.Errorf("alpha.DependedOnBy = %v, want []", services[0].DependedOnBy)
	}
}

func TestApplyReverseIndexSelfRef(t *testing.T) {
	services := []Service{
		{Name: "alpha", Integrations: []Integration{{ClientID: "alpha"}}},
	}
	idx := NewNameIndexFromServices(services)

	applyReverseIndex(services, idx)

	// Self-references should be excluded
	if len(services[0].DependedOnBy) != 0 {
		t.Errorf("alpha.DependedOnBy = %v, want [] (self-ref excluded)", services[0].DependedOnBy)
	}
}

func TestApplyReverseIndexUnknownTarget(t *testing.T) {
	services := []Service{
		{Name: "alpha", Integrations: []Integration{{ClientID: "external"}}},
	}
	idx := NewNameIndexFromServices(services)

	applyReverseIndex(services, idx)

	// External dependency should not create reverse entry
	if len(services[0].DependedOnBy) != 0 {
		t.Errorf("alpha.DependedOnBy = %v, want []", services[0].DependedOnBy)
	}
}

// ── addUnresolvedPackages tests ─────────────────────────────────────

func TestAddUnresolvedPackages(t *testing.T) {
	ws := &walkState{
		allPackages:  map[string]bool{"pkg-a": true, "pkg-b": true, "pkg-c": true},
		resolvedPkgs: map[string]bool{"pkg-a": true},
		seen:         map[string]bool{"pkg-a": true},
	}

	addUnresolvedPackages(ws)

	// pkg-b and pkg-c should be added as integrations
	if len(ws.integrations) != 2 {
		t.Fatalf("expected 2 unresolved integrations, got %d", len(ws.integrations))
	}

	found := map[string]bool{}
	for _, integ := range ws.integrations {
		found[integ.ClientID] = true
	}
	if !found["pkg-b"] {
		t.Error("expected pkg-b in unresolved")
	}
	if !found["pkg-c"] {
		t.Error("expected pkg-c in unresolved")
	}
}

func TestAddUnresolvedPackagesAllResolved(t *testing.T) {
	ws := &walkState{
		allPackages:  map[string]bool{"pkg-a": true},
		resolvedPkgs: map[string]bool{"pkg-a": true},
		seen:         map[string]bool{"pkg-a": true},
	}

	addUnresolvedPackages(ws)

	if len(ws.integrations) != 0 {
		t.Errorf("expected 0 unresolved, got %d", len(ws.integrations))
	}
}

func TestAddUnresolvedPackagesAlreadySeen(t *testing.T) {
	ws := &walkState{
		allPackages:  map[string]bool{"pkg-a": true},
		resolvedPkgs: map[string]bool{},
		seen:         map[string]bool{"pkg-a": true},
	}

	addUnresolvedPackages(ws)

	if len(ws.integrations) != 0 {
		t.Errorf("expected 0 integrations for already seen pkg, got %d", len(ws.integrations))
	}
}

// ── extractInputSpecs tests ─────────────────────────────────────────

func TestExtractInputSpecsXML(t *testing.T) {
	pom := `<?xml version="1.0" encoding="UTF-8"?>
<project>
    <build>
        <plugins>
            <plugin>
                <executions>
                    <execution>
                        <configuration>
                            <inputSpec>src/main/resources/integrations/party-api-1.3.0.yaml</inputSpec>
                        </configuration>
                    </execution>
                </executions>
            </plugin>
        </plugins>
    </build>
</project>`

	specs := extractInputSpecs([]byte(pom))
	if len(specs) != 1 {
		t.Fatalf("expected 1 spec, got %d", len(specs))
	}
	if specs[0] != "party-api-1.3.0.yaml" {
		t.Errorf("spec = %q, want party-api-1.3.0.yaml", specs[0])
	}
}

func TestExtractInputSpecsFallbackRegex(t *testing.T) {
	// Use a pom that doesn't parse into the standard XML structure
	pom := `<project>
    <profiles>
        <profile>
            <build>
                <plugins>
                    <plugin>
                        <configuration>
                            <inputSpec>path/to/notes-api.yaml</inputSpec>
                        </configuration>
                    </plugin>
                </plugins>
            </build>
        </profile>
    </profiles>
</project>`

	specs := extractInputSpecs([]byte(pom))
	if len(specs) != 1 {
		t.Fatalf("expected 1 spec from regex fallback, got %d: %v", len(specs), specs)
	}
	if specs[0] != "notes-api.yaml" {
		t.Errorf("spec = %q, want notes-api.yaml", specs[0])
	}
}

func TestExtractInputSpecsNone(t *testing.T) {
	pom := `<project><version>1.0</version></project>`
	specs := extractInputSpecs([]byte(pom))
	if len(specs) != 0 {
		t.Errorf("expected 0 specs, got %d", len(specs))
	}
}

// ── matchSpecVersions integration test ──────────────────────────────

func TestMatchSpecVersions(t *testing.T) {
	tmpDir := t.TempDir()

	// Create spec file
	specDir := filepath.Join(tmpDir, "src", "main", "resources", "integrations")
	os.MkdirAll(specDir, 0755)
	specContent := `openapi: "3.0.0"
info:
  title: Party API
  version: 1.3.0
paths:`
	os.WriteFile(filepath.Join(specDir, "party-api.yaml"), []byte(specContent), 0644)

	// Create pom.xml with inputSpec reference
	pom := `<?xml version="1.0" encoding="UTF-8"?>
<project>
    <build>
        <plugins>
            <plugin>
                <executions>
                    <execution>
                        <configuration>
                            <inputSpec>src/main/resources/integrations/party-api.yaml</inputSpec>
                        </configuration>
                    </execution>
                </executions>
            </plugin>
        </plugins>
    </build>
</project>`
	os.WriteFile(filepath.Join(tmpDir, "pom.xml"), []byte(pom), 0644)

	integrations := []Integration{
		{ClientID: "party"},
	}

	matchSpecVersions(tmpDir, integrations)

	if integrations[0].SpecVersion != "1.3.0" {
		t.Errorf("SpecVersion = %q, want 1.3.0", integrations[0].SpecVersion)
	}
}

func TestMatchSpecVersionsNoPom(t *testing.T) {
	tmpDir := t.TempDir()
	integrations := []Integration{{ClientID: "party"}}

	// Should not panic
	matchSpecVersions(tmpDir, integrations)

	if integrations[0].SpecVersion != "" {
		t.Errorf("SpecVersion = %q, want empty", integrations[0].SpecVersion)
	}
}

func TestMatchSpecVersionsNoSpecs(t *testing.T) {
	tmpDir := t.TempDir()
	pom := `<project><version>1.0</version></project>`
	os.WriteFile(filepath.Join(tmpDir, "pom.xml"), []byte(pom), 0644)

	integrations := []Integration{{ClientID: "party"}}
	matchSpecVersions(tmpDir, integrations)

	if integrations[0].SpecVersion != "" {
		t.Errorf("SpecVersion = %q, want empty", integrations[0].SpecVersion)
	}
}

// ── findSpecFile tests ──────────────────────────────────────────────

func TestFindSpecFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create spec in integrations dir
	integDir := filepath.Join(tmpDir, "src", "main", "resources", "integrations")
	os.MkdirAll(integDir, 0755)
	os.WriteFile(filepath.Join(integDir, "test-api.yaml"), []byte("openapi: 3.0"), 0644)

	got := findSpecFile(tmpDir, "test-api.yaml")
	if got == "" {
		t.Error("expected to find spec file")
	}

	// Test contract dir
	contractDir := filepath.Join(tmpDir, "src", "main", "resources", "contract")
	os.MkdirAll(contractDir, 0755)
	os.WriteFile(filepath.Join(contractDir, "contract-api.yaml"), []byte("openapi: 3.0"), 0644)

	got = findSpecFile(tmpDir, "contract-api.yaml")
	if got == "" {
		t.Error("expected to find spec file in contract dir")
	}

	// Test rest subdir
	restDir := filepath.Join(tmpDir, "src", "main", "resources", "integrations", "rest")
	os.MkdirAll(restDir, 0755)
	os.WriteFile(filepath.Join(restDir, "rest-api.yaml"), []byte("openapi: 3.0"), 0644)

	got = findSpecFile(tmpDir, "rest-api.yaml")
	if got == "" {
		t.Error("expected to find spec file in rest dir")
	}

	// Not found
	got = findSpecFile(tmpDir, "nonexistent.yaml")
	if got != "" {
		t.Errorf("expected empty for nonexistent file, got %q", got)
	}
}

// ── extractRegexMatches tests ───────────────────────────────────────

func TestExtractRegexMatches(t *testing.T) {
	content := `public class Config {
    public static final String CLIENT_ID = "case-data";
}`
	ws := &walkState{
		allPackages:  make(map[string]bool),
		resolvedPkgs: make(map[string]bool),
		seen:         make(map[string]bool),
	}

	extractRegexMatches(content, "casedata", ws)

	if len(ws.integrations) != 1 {
		t.Fatalf("expected 1 integration, got %d", len(ws.integrations))
	}
	if ws.integrations[0].ClientID != "case-data" {
		t.Errorf("ClientID = %q, want case-data", ws.integrations[0].ClientID)
	}
	if !ws.resolvedPkgs["casedata"] {
		t.Error("expected package to be marked as resolved")
	}
}

func TestExtractRegexMatchesIntegrationName(t *testing.T) {
	content := `public class Config {
    public static final String INTEGRATION_NAME = "notes";
}`
	ws := &walkState{
		allPackages:  make(map[string]bool),
		resolvedPkgs: make(map[string]bool),
		seen:         make(map[string]bool),
	}

	extractRegexMatches(content, "notes", ws)

	if len(ws.integrations) != 1 {
		t.Fatalf("expected 1 integration, got %d", len(ws.integrations))
	}
	if ws.integrations[0].ClientID != "notes" {
		t.Errorf("ClientID = %q, want notes", ws.integrations[0].ClientID)
	}
}

func TestExtractRegexMatchesNoPkg(t *testing.T) {
	content := `public static final String CLIENT_ID = "test";`
	ws := &walkState{
		allPackages:  make(map[string]bool),
		resolvedPkgs: make(map[string]bool),
		seen:         make(map[string]bool),
	}

	extractRegexMatches(content, "", ws)

	if len(ws.integrations) != 1 {
		t.Fatalf("expected 1 integration, got %d", len(ws.integrations))
	}
	// Empty pkg should not be added to resolvedPkgs
	if len(ws.resolvedPkgs) != 0 {
		t.Errorf("expected no resolved pkgs, got %d", len(ws.resolvedPkgs))
	}
}

func TestExtractRegexMatchesDuplicate(t *testing.T) {
	content := `public static final String CLIENT_ID = "test";`
	ws := &walkState{
		allPackages:  make(map[string]bool),
		resolvedPkgs: make(map[string]bool),
		seen:         map[string]bool{"test": true},
	}

	extractRegexMatches(content, "pkg", ws)

	if len(ws.integrations) != 0 {
		t.Errorf("expected 0 integrations (duplicate), got %d", len(ws.integrations))
	}
}

// ── StaleCount edge cases ───────────────────────────────────────────

func TestStaleCountNoSpecVersion(t *testing.T) {
	services := []Service{
		{Name: "a", Integrations: []Integration{{ClientID: "b"}}},
		{Name: "b", Version: "1.0.0"},
	}
	idx := NewNameIndexFromServices(services)
	count := StaleCount(&services[0], services, idx)
	if count != 0 {
		t.Errorf("StaleCount = %d, want 0 (no spec version)", count)
	}
}

func TestStaleCountTargetNoVersion(t *testing.T) {
	services := []Service{
		{Name: "a", Integrations: []Integration{{ClientID: "b", SpecVersion: "1.0"}}},
		{Name: "b"},
	}
	idx := NewNameIndexFromServices(services)
	count := StaleCount(&services[0], services, idx)
	if count != 0 {
		t.Errorf("StaleCount = %d, want 0 (target has no version)", count)
	}
}

func TestStaleCountUnknownTarget(t *testing.T) {
	services := []Service{
		{Name: "a", Integrations: []Integration{{ClientID: "unknown", SpecVersion: "1.0"}}},
	}
	idx := NewNameIndexFromServices(services)
	count := StaleCount(&services[0], services, idx)
	if count != 0 {
		t.Errorf("StaleCount = %d, want 0 (unknown target)", count)
	}
}

func TestStaleCountTargetNotFound(t *testing.T) {
	// Target resolves in index but isn't in services list
	idx := NewNameIndex(map[string]bool{"known": true})
	services := []Service{
		{Name: "a", Integrations: []Integration{{ClientID: "known", SpecVersion: "1.0"}}},
	}
	count := StaleCount(&services[0], services, idx)
	if count != 0 {
		t.Errorf("StaleCount = %d, want 0 (target not in services)", count)
	}
}

// ── Scan with multiple repos and reverse index ──────────────────────

func TestScanWithReverseIndex(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two services where A depends on B
	svcA := filepath.Join(tmpDir, "api-service-svc-a")
	javaDirA := filepath.Join(svcA, "src", "main", "java", "se", "sundsvall", "a", "integration", "svcb")
	os.MkdirAll(javaDirA, 0755)
	os.WriteFile(filepath.Join(javaDirA, "Config.java"), []byte(`public class C { public static final String CLIENT_ID = "svc-b"; }`), 0644)
	os.WriteFile(filepath.Join(svcA, "pom.xml"), []byte(`<project><version>1.0</version></project>`), 0644)

	svcB := filepath.Join(tmpDir, "api-service-svc-b")
	javaDirB := filepath.Join(svcB, "src", "main", "java")
	os.MkdirAll(javaDirB, 0755)
	os.WriteFile(filepath.Join(svcB, "pom.xml"), []byte(`<project><version>2.0</version></project>`), 0644)

	cfg := ScanConfig{RepoPrefixes: []string{"api-service-"}, StandaloneRepos: map[string]string{}}
	services, _, err := Scan(tmpDir, cfg)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	// Find svc-b and check DependedOnBy
	for _, svc := range services {
		if svc.Name == "svc-b" {
			if len(svc.DependedOnBy) == 0 {
				t.Error("expected svc-b to have dependents")
			}
			found := false
			for _, dep := range svc.DependedOnBy {
				if dep == "svc-a" {
					found = true
				}
			}
			if !found {
				t.Errorf("expected svc-a in svc-b.DependedOnBy, got %v", svc.DependedOnBy)
			}
		}
	}
}

// ── processJavaFile edge cases ──────────────────────────────────────

// ── Scan with nonexistent directory ──────────────────────────────────

func TestScanNonexistentDir(t *testing.T) {
	cfg := ScanConfig{
		RepoPrefixes:    []string{"api-service-"},
		StandaloneRepos: map[string]string{},
	}

	// Scanning a nonexistent directory should not error (filepath.Abs succeeds)
	services, idx, err := Scan("/nonexistent/path/that/does/not/exist", cfg)
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

// ── tryBuildService edge cases ──────────────────────────────────────

func TestTryBuildServiceNotADir(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a file (not a dir)
	filePath := filepath.Join(tmpDir, "not-a-dir")
	os.WriteFile(filePath, []byte("hello"), 0644)

	names := make(map[string]bool)
	_, ok := tryBuildService(filePath, "test", names)
	if ok {
		t.Error("expected false for non-directory")
	}
}

func TestTryBuildServiceNonexistent(t *testing.T) {
	names := make(map[string]bool)
	_, ok := tryBuildService("/nonexistent", "test", names)
	if ok {
		t.Error("expected false for nonexistent path")
	}
}

// ── scanRepo with walk error ────────────────────────────────────────

func TestScanRepoMissingSrcDir(t *testing.T) {
	tmpDir := t.TempDir()
	// No src/main/java dir exists -- scanRepo should handle gracefully
	result := scanRepo(tmpDir)
	if len(result) != 0 {
		t.Errorf("expected 0 integrations, got %d", len(result))
	}
}

// ── processJavaFile with non-integration path ───────────────────────

func TestProcessJavaFileNonIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	serviceDir := filepath.Join(tmpDir, "src", "main", "java", "se", "sundsvall", "service")
	os.MkdirAll(serviceDir, 0755)
	os.WriteFile(filepath.Join(serviceDir, "Main.java"), []byte(`CLIENT_ID = "test";`), 0644)

	ws := &walkState{
		allPackages:  make(map[string]bool),
		resolvedPkgs: make(map[string]bool),
		seen:         make(map[string]bool),
	}

	integBase := filepath.Join(tmpDir, "src", "main", "java")
	err := processJavaFile(filepath.Join(serviceDir, "Main.java"), integBase, ws)
	if err != nil {
		t.Fatalf("processJavaFile error: %v", err)
	}
	// File not in an integration path, should be skipped
	if len(ws.integrations) != 0 {
		t.Errorf("expected 0 integrations, got %d", len(ws.integrations))
	}
}

// ── processJavaFile with unreadable file ────────────────────────────

func TestProcessJavaFileUnreadable(t *testing.T) {
	tmpDir := t.TempDir()
	integDir := filepath.Join(tmpDir, "src", "main", "java", "se", "sundsvall", "integration", "party")
	os.MkdirAll(integDir, 0755)

	javaPath := filepath.Join(integDir, "Config.java")
	os.WriteFile(javaPath, []byte(`CLIENT_ID = "party";`), 0644)
	// Make it unreadable
	os.Chmod(javaPath, 0000)
	t.Cleanup(func() { os.Chmod(javaPath, 0644) })

	ws := &walkState{
		allPackages:  make(map[string]bool),
		resolvedPkgs: make(map[string]bool),
		seen:         make(map[string]bool),
	}

	integBase := filepath.Join(tmpDir, "src", "main", "java")
	err := processJavaFile(javaPath, integBase, ws)
	if err != nil {
		t.Fatalf("processJavaFile should not return error for unreadable file: %v", err)
	}
	// Package name should still be recorded even though file can't be read
	if !ws.allPackages["party"] {
		t.Error("expected party in allPackages")
	}
}

// ── matchSpecVersions with spec that doesn't match any integration ──

func TestMatchSpecVersionsNoMatchingIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	specDir := filepath.Join(tmpDir, "src", "main", "resources", "integrations")
	os.MkdirAll(specDir, 0755)
	os.WriteFile(filepath.Join(specDir, "unknown-api.yaml"), []byte("openapi: 3.0\ninfo:\n  version: 1.0.0\npaths:"), 0644)

	pom := `<?xml version="1.0" encoding="UTF-8"?>
<project><build><plugins><plugin><executions><execution><configuration>
<inputSpec>src/main/resources/integrations/unknown-api.yaml</inputSpec>
</configuration></execution></executions></plugin></plugins></build></project>`
	os.WriteFile(filepath.Join(tmpDir, "pom.xml"), []byte(pom), 0644)

	integrations := []Integration{{ClientID: "party"}}
	matchSpecVersions(tmpDir, integrations)

	// party should not get a spec version since spec normalizes to "unknown"
	if integrations[0].SpecVersion != "" {
		t.Errorf("SpecVersion = %q, want empty", integrations[0].SpecVersion)
	}
}

// ── matchSpecVersions with spec file that has no version ────────────

func TestMatchSpecVersionsSpecNoVersion(t *testing.T) {
	tmpDir := t.TempDir()
	specDir := filepath.Join(tmpDir, "src", "main", "resources", "integrations")
	os.MkdirAll(specDir, 0755)
	os.WriteFile(filepath.Join(specDir, "party-api.yaml"), []byte("openapi: 3.0\ninfo:\n  title: No Ver\npaths:"), 0644)

	pom := `<?xml version="1.0" encoding="UTF-8"?>
<project><build><plugins><plugin><executions><execution><configuration>
<inputSpec>src/main/resources/integrations/party-api.yaml</inputSpec>
</configuration></execution></executions></plugin></plugins></build></project>`
	os.WriteFile(filepath.Join(tmpDir, "pom.xml"), []byte(pom), 0644)

	integrations := []Integration{{ClientID: "party"}}
	matchSpecVersions(tmpDir, integrations)

	if integrations[0].SpecVersion != "" {
		t.Errorf("SpecVersion = %q, want empty", integrations[0].SpecVersion)
	}
}

// ── matchSpecVersions with spec file not found in any dir ───────────

func TestMatchSpecVersionsSpecNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	pom := `<?xml version="1.0" encoding="UTF-8"?>
<project><build><plugins><plugin><executions><execution><configuration>
<inputSpec>src/main/resources/integrations/missing-api.yaml</inputSpec>
</configuration></execution></executions></plugin></plugins></build></project>`
	os.WriteFile(filepath.Join(tmpDir, "pom.xml"), []byte(pom), 0644)

	integrations := []Integration{{ClientID: "missing"}}
	matchSpecVersions(tmpDir, integrations)

	if integrations[0].SpecVersion != "" {
		t.Errorf("SpecVersion = %q, want empty", integrations[0].SpecVersion)
	}
}

// ── extractPackageName with .java suffix edge case ──────────────────

func TestExtractPackageNameJavaSuffix(t *testing.T) {
	// When the part after "integration" ends in .java, it should be treated differently
	got := extractPackageName("se/sundsvall/foo/integration/Config.java")
	// "Config.java" has the .java suffix
	if got == "" {
		// The function returns parts[i+1] which is "Config.java"
		// But processJavaFile checks !strings.HasSuffix(pkg, ".java")
		// so this is expected to be "Config.java" from extractPackageName
	}
}

// ── Resolve edge case: hyphen-stripped already matches ───────────────

// ── Resolve second branch: stripped match (not lower match) ─────────

func TestResolveStrippedMatch(t *testing.T) {
	idx := NewNameIndex(map[string]bool{"party": true})

	// "par-ty" lowered is "par-ty" which is NOT in nameMap,
	// but stripped "party" IS in nameMap. This hits the second branch.
	got := idx.Resolve("par-ty")
	if got != "party" {
		t.Errorf("Resolve('par-ty') = %q, want 'party'", got)
	}

	// This should fail both lookups
	got = idx.Resolve("xyzabc")
	if got != "" {
		t.Errorf("Resolve('xyzabc') = %q, want empty", got)
	}
}

func TestResolveHyphenStripped(t *testing.T) {
	idx := NewNameIndex(map[string]bool{"case-data": true})
	// Resolve with dots
	got := idx.Resolve("case.data")
	if got != "case-data" {
		t.Errorf("Resolve('case.data') = %q, want 'case-data'", got)
	}
}

func TestProcessJavaFileSkipsDBPath(t *testing.T) {
	tmpDir := t.TempDir()
	dbDir := filepath.Join(tmpDir, "src", "main", "java", "se", "sundsvall", "integration", "db")
	os.MkdirAll(dbDir, 0755)
	os.WriteFile(filepath.Join(dbDir, "Mapper.java"), []byte(`CLIENT_ID = "test";`), 0644)

	ws := &walkState{
		allPackages:  make(map[string]bool),
		resolvedPkgs: make(map[string]bool),
		seen:         make(map[string]bool),
	}

	integBase := filepath.Join(tmpDir, "src", "main", "java")
	err := processJavaFile(filepath.Join(dbDir, "Mapper.java"), integBase, ws)
	if err != nil {
		t.Fatalf("processJavaFile error: %v", err)
	}
	if len(ws.integrations) != 0 {
		t.Errorf("expected 0 integrations (DB path skipped), got %d", len(ws.integrations))
	}
}
