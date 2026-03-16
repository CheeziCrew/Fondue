package scanner

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Integration struct {
	ClientID    string // e.g. "case-data"
	SpecVersion string // version from integration spec's info.version (empty if no spec found)
}

type Service struct {
	Name         string        // e.g. "case-data" (from dir name)
	Path         string        // absolute path to repo
	Version      string        // from pom.xml <version>
	Integrations []Integration // outbound dependencies
	DependedOnBy []string      // computed: which services integrate with this one
}

// ScanConfig holds the configurable parts of scanning.
type ScanConfig struct {
	RepoPrefixes    []string
	StandaloneRepos map[string]string
}

// NameIndex is a pre-computed lookup for resolving client IDs to service names.
// Built once after scanning, reused everywhere.
type NameIndex struct {
	nameMap map[string]string
	aliases map[string][]string // collision aliases: stripped name → all full names
}

var (
	clientIDRegex        = regexp.MustCompile(`CLIENT_ID\s*=\s*"([^"]+)"`)
	integrationNameRegex = regexp.MustCompile(`INTEGRATION_NAME\s*=\s*"([^"]+)"`)
	specFileCleanRegex   = regexp.MustCompile(`(?i)(-api)?(-(v?\d+(\.\d+)*))?\.ya?ml$`)
)

// ── XML structs for proper pom.xml parsing ──────────────────────────

type pomProject struct {
	XMLName xml.Name  `xml:"project"`
	Version string    `xml:"version"`
	Build   *pomBuild `xml:"build"`
}

type pomBuild struct {
	Plugins pomPlugins `xml:"plugins"`
}

type pomPlugins struct {
	Plugins []pomPlugin `xml:"plugin"`
}

type pomPlugin struct {
	Executions pomExecutions `xml:"executions"`
}

type pomExecutions struct {
	Executions []pomExecution `xml:"execution"`
}

type pomExecution struct {
	Configuration pomConfiguration `xml:"configuration"`
}

type pomConfiguration struct {
	InputSpec string `xml:"inputSpec"`
}

func Scan(basePath string, cfg ScanConfig) ([]Service, *NameIndex, error) {
	basePath, err := filepath.Abs(basePath)
	if err != nil {
		return nil, nil, err
	}

	services := make([]Service, 0, 80)
	serviceNames := make(map[string]bool)

	var collisionAliases map[string][]string
	services, collisionAliases, err = scanPrefixedRepos(basePath, cfg.RepoPrefixes, services, serviceNames)
	if err != nil {
		return nil, nil, err
	}

	services = scanStandaloneRepos(basePath, cfg.StandaloneRepos, services, serviceNames)

	idx := NewNameIndex(serviceNames)
	idx.addAliases(collisionAliases)
	normalizeClientIDs(services, idx)
	applyReverseIndex(services, idx)

	sort.Slice(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})

	return services, idx, nil
}

func scanPrefixedRepos(basePath string, prefixes []string, services []Service, serviceNames map[string]bool) ([]Service, map[string][]string, error) {
	type candidate struct {
		dir          string
		strippedName string
		fullName     string
	}
	var candidates []candidate
	nameCount := make(map[string]int)

	for _, prefix := range prefixes {
		dirs, err := filepath.Glob(filepath.Join(basePath, prefix+"*"))
		if err != nil {
			return nil, nil, err
		}
		for _, dir := range dirs {
			fullName := filepath.Base(dir)
			stripped := strings.TrimPrefix(fullName, prefix)
			candidates = append(candidates, candidate{dir, stripped, fullName})
			nameCount[stripped]++
		}
	}

	aliases := make(map[string][]string)
	for _, c := range candidates {
		name := c.strippedName
		if nameCount[c.strippedName] > 1 {
			name = c.fullName
		}
		svc, ok := tryBuildService(c.dir, name, serviceNames)
		if ok {
			services = append(services, svc)
			if nameCount[c.strippedName] > 1 {
				aliases[c.strippedName] = append(aliases[c.strippedName], name)
			}
		}
	}

	return services, aliases, nil
}

func scanStandaloneRepos(basePath string, repos map[string]string, services []Service, serviceNames map[string]bool) []Service {
	for dirName, serviceName := range repos {
		dir := filepath.Join(basePath, dirName)
		svc, ok := tryBuildService(dir, serviceName, serviceNames)
		if ok {
			services = append(services, svc)
		}
	}
	return services
}

func tryBuildService(dir, name string, serviceNames map[string]bool) (Service, bool) {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return Service{}, false
	}
	if _, err := os.Stat(filepath.Join(dir, "src", "main", "java")); err != nil {
		return Service{}, false
	}
	if serviceNames[name] {
		return Service{}, false
	}
	serviceNames[name] = true

	integrations := scanRepo(dir)
	matchSpecVersions(dir, integrations)
	return Service{
		Name:         name,
		Path:         dir,
		Version:      extractPomVersion(dir),
		Integrations: integrations,
	}, true
}

func normalizeClientIDs(services []Service, idx *NameIndex) {
	for i := range services {
		for j := range services[i].Integrations {
			normalizeClientID(&services[i].Integrations[j], idx, services[i].Name)
		}
	}
}

func normalizeClientID(integ *Integration, idx *NameIndex, selfName string) {
	if r := idx.ResolveExcluding(integ.ClientID, selfName); r != "" {
		return
	}
	lower := strings.ToLower(integ.ClientID)
	for _, suffix := range []string{"client", "integration", "service", "process"} {
		trimmed := strings.TrimSuffix(lower, suffix)
		if trimmed != lower {
			if resolved := idx.ResolveExcluding(trimmed, selfName); resolved != "" {
				integ.ClientID = resolved
				return
			}
		}
	}
}

func applyReverseIndex(services []Service, idx *NameIndex) {
	reverseIndex := make(map[string][]string)
	for i := range services {
		for _, integ := range services[i].Integrations {
			target := idx.Resolve(integ.ClientID)
			if target != "" && target != services[i].Name {
				reverseIndex[target] = appendUnique(reverseIndex[target], services[i].Name)
			}
		}
	}

	for i := range services {
		if deps, ok := reverseIndex[services[i].Name]; ok {
			sort.Strings(deps)
			services[i].DependedOnBy = deps
		}
	}
}

// ── NameIndex ──────────────────────────────────────────────────────

func NewNameIndex(serviceNames map[string]bool) *NameIndex {
	m := make(map[string]string)
	for name := range serviceNames {
		m[name] = name
		stripped := strings.ReplaceAll(name, "-", "")
		m[stripped] = name
		dotted := strings.ReplaceAll(name, "-", ".")
		m[dotted] = name
		if strings.HasSuffix(name, "s") {
			m[strings.TrimSuffix(name, "s")] = name
			m[strings.TrimSuffix(stripped, "s")] = name
		} else {
			m[name+"s"] = name
			m[stripped+"s"] = name
		}
	}
	return &NameIndex{nameMap: m}
}

// NewNameIndexFromServices builds a NameIndex from a slice of services.
func NewNameIndexFromServices(services []Service) *NameIndex {
	names := make(map[string]bool, len(services))
	for _, s := range services {
		names[s.Name] = true
	}
	return NewNameIndex(names)
}

func (idx *NameIndex) addAliases(aliasGroups map[string][]string) {
	if len(aliasGroups) == 0 {
		return
	}
	idx.aliases = make(map[string][]string)
	for alias, names := range aliasGroups {
		idx.aliases[alias] = names
		stripped := strings.ReplaceAll(alias, "-", "")
		if stripped != alias {
			idx.aliases[stripped] = names
		}
	}
}

func (idx *NameIndex) Resolve(clientID string) string {
	lower := strings.ToLower(clientID)
	if name, ok := idx.nameMap[lower]; ok {
		return name
	}
	stripped := strings.ReplaceAll(lower, "-", "")
	if name, ok := idx.nameMap[stripped]; ok {
		return name
	}
	return ""
}

// ResolveExcluding resolves a client ID but skips the excluded name.
// Falls back to collision aliases when the primary match is excluded.
func (idx *NameIndex) ResolveExcluding(clientID, exclude string) string {
	result := idx.Resolve(clientID)
	if result != "" && result != exclude {
		return result
	}
	if idx.aliases != nil {
		lower := strings.ToLower(clientID)
		for _, key := range []string{lower, strings.ReplaceAll(lower, "-", "")} {
			if names, ok := idx.aliases[key]; ok {
				for _, n := range names {
					if n != exclude {
						return n
					}
				}
			}
		}
	}
	return result
}

func (idx *NameIndex) IsInternal(clientID string) bool {
	return idx.Resolve(clientID) != ""
}

// ── Repo scanning ──────────────────────────────────────────────────

type walkState struct {
	allPackages  map[string]bool
	resolvedPkgs map[string]bool
	seen         map[string]bool
	integrations []Integration
}

func scanRepo(repoPath string) []Integration {
	integrationBase := filepath.Join(repoPath, "src", "main", "java")

	ws := &walkState{
		allPackages:  make(map[string]bool),
		resolvedPkgs: make(map[string]bool),
		seen:         make(map[string]bool),
	}

	err := filepath.Walk(integrationBase, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".java") {
			return nil
		}
		return processJavaFile(path, integrationBase, ws)
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: error scanning %s: %v\n", repoPath, err)
	}

	addUnresolvedPackages(ws)

	sort.Slice(ws.integrations, func(i, j int) bool {
		return ws.integrations[i].ClientID < ws.integrations[j].ClientID
	})

	return ws.integrations
}

func processJavaFile(path, integrationBase string, ws *walkState) error {
	rel, _ := filepath.Rel(integrationBase, path)
	if !strings.Contains(rel, "integration") {
		return nil
	}
	if isDBPath(rel) {
		return nil
	}

	pkg := extractPackageName(rel)
	if pkg != "" && !strings.HasSuffix(pkg, ".java") {
		ws.allPackages[pkg] = true
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	extractRegexMatches(string(data), pkg, ws)
	return nil
}

func isDBPath(rel string) bool {
	parts := strings.Split(rel, string(filepath.Separator))
	for _, p := range parts {
		if p == "db" {
			return true
		}
	}
	return false
}

func extractRegexMatches(content, pkg string, ws *walkState) {
	if matches := clientIDRegex.FindStringSubmatch(content); len(matches) > 1 {
		addIntegration(&ws.integrations, ws.seen, matches[1])
		if pkg != "" {
			ws.resolvedPkgs[pkg] = true
		}
	}
	if matches := integrationNameRegex.FindStringSubmatch(content); len(matches) > 1 {
		addIntegration(&ws.integrations, ws.seen, matches[1])
		if pkg != "" {
			ws.resolvedPkgs[pkg] = true
		}
	}
}

func addUnresolvedPackages(ws *walkState) {
	for pkg := range ws.allPackages {
		if !ws.resolvedPkgs[pkg] && !ws.seen[pkg] {
			ws.seen[pkg] = true
			ws.integrations = append(ws.integrations, Integration{ClientID: pkg})
		}
	}
}

func extractPackageName(relPath string) string {
	parts := strings.Split(relPath, string(filepath.Separator))
	for i, p := range parts {
		if p == "integration" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func addIntegration(integrations *[]Integration, seen map[string]bool, clientID string) {
	if !seen[clientID] {
		seen[clientID] = true
		*integrations = append(*integrations, Integration{ClientID: clientID})
	}
}

// ── Version extraction (proper XML parsing) ────────────────────────

func extractPomVersion(repoPath string) string {
	data, err := os.ReadFile(filepath.Join(repoPath, "pom.xml"))
	if err != nil {
		return ""
	}

	var pom pomProject
	if xml.Unmarshal(data, &pom) != nil {
		return ""
	}

	if v := strings.TrimSpace(pom.Version); v != "" && v != "@project.version@" {
		return v
	}
	return ""
}

// ── Spec version matching ──────────────────────────────────────────

func matchSpecVersions(repoPath string, integrations []Integration) {
	data, err := os.ReadFile(filepath.Join(repoPath, "pom.xml"))
	if err != nil {
		return
	}

	// Extract inputSpec paths from pom.xml using XML parsing
	specFiles := extractInputSpecs(data)
	if len(specFiles) == 0 {
		return
	}

	specVersions := make(map[string]string)
	for _, specFilename := range specFiles {
		specPath := findSpecFile(repoPath, specFilename)
		if specPath == "" {
			continue
		}

		version := extractSpecVersion(specPath)
		if version == "" {
			continue
		}

		normalized := normalizeSpecFilename(specFilename)
		specVersions[normalized] = version
	}

	for i := range integrations {
		normalizedID := strings.ToLower(strings.ReplaceAll(integrations[i].ClientID, "-", ""))
		for specName, version := range specVersions {
			if specName == normalizedID {
				integrations[i].SpecVersion = version
				break
			}
		}
	}
}

// extractInputSpecs pulls inputSpec filenames from pom.xml via XML parsing.
// Falls back to regex if XML structure doesn't match (some poms use profiles/pluginManagement).
func extractInputSpecs(pomData []byte) []string {
	var pom pomProject
	if err := xml.Unmarshal(pomData, &pom); err == nil && pom.Build != nil {
		var specs []string
		for _, plugin := range pom.Build.Plugins.Plugins {
			for _, exec := range plugin.Executions.Executions {
				if spec := exec.Configuration.InputSpec; spec != "" {
					// Extract just the filename
					specs = append(specs, filepath.Base(spec))
				}
			}
		}
		if len(specs) > 0 {
			return specs
		}
	}

	// Fallback: regex for poms with non-standard structure
	re := regexp.MustCompile(`<inputSpec>[^<]*?([^/<]+\.ya?ml)</inputSpec>`)
	matches := re.FindAllStringSubmatch(string(pomData), -1)
	var specs []string
	for _, m := range matches {
		specs = append(specs, m[1])
	}
	return specs
}

func findSpecFile(repoPath, filename string) string {
	dirs := []string{
		"src/main/resources/integrations",
		"src/main/resources/integrations/rest",
		"src/main/resources/contract",
	}
	for _, dir := range dirs {
		path := filepath.Join(repoPath, dir, filename)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// extractSpecVersion reads info.version from an OpenAPI spec YAML header.
// Only reads first 15 lines to stay fast.
func extractSpecVersion(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	lines := strings.SplitN(string(data), "\n", 16)
	re := regexp.MustCompile(`(?m)^\s+version:\s*["']?([^"'\n\r]+)["']?\s*$`)
	header := strings.Join(lines, "\n")

	if m := re.FindStringSubmatch(header); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func normalizeSpecFilename(filename string) string {
	name := specFileCleanRegex.ReplaceAllString(filename, "")
	name = strings.TrimPrefix(name, "api-")
	return strings.ToLower(strings.ReplaceAll(name, "-", ""))
}

func appendUnique(slice []string, val string) []string {
	for _, s := range slice {
		if s == val {
			return slice
		}
	}
	return append(slice, val)
}

// ── Public helpers (use NameIndex) ─────────────────────────────────

func FindService(name string, services []Service) *Service {
	for i := range services {
		if services[i].Name == name {
			return &services[i]
		}
	}
	return nil
}

func StaleCount(svc *Service, services []Service, idx *NameIndex) int {
	count := 0
	for _, integ := range svc.Integrations {
		if integ.SpecVersion == "" {
			continue
		}
		targetName := idx.Resolve(integ.ClientID)
		if targetName == "" {
			continue
		}
		target := FindService(targetName, services)
		if target == nil || target.Version == "" {
			continue
		}
		if integ.SpecVersion != target.Version {
			count++
		}
	}
	return count
}
