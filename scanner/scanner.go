package scanner

import (
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

// Repo prefixes to scan, in order. The prefix is stripped to derive the service name.
// More specific prefixes first — a repo matched by an earlier prefix won't be re-scanned.
var repoPrefixes = []string{
	"api-service-",
	"pw-",
}

// Standalone repos that don't follow standard prefixes but are real services.
var standaloneRepos = map[string]string{
	"api-comfact-facade": "comfact-facade",
	"cimd-proxy":         "cimd-proxy",
	"formpipe-proxy":     "formpipe-proxy",
}

var (
	clientIDRegex        = regexp.MustCompile(`CLIENT_ID\s*=\s*"([^"]+)"`)
	integrationNameRegex = regexp.MustCompile(`INTEGRATION_NAME\s*=\s*"([^"]+)"`)
	pomVersionRegex      = regexp.MustCompile(`<version>([^<]+)</version>`)
	inputSpecRegex       = regexp.MustCompile(`<inputSpec>[^<]*?([^/<]+\.ya?ml)</inputSpec>`)
	specVersionRegex     = regexp.MustCompile(`(?m)^\s+version:\s*["']?([^"'\n\r]+)["']?\s*$`)
	specFileCleanRegex   = regexp.MustCompile(`(?i)(-api|-(v?\d+(\.\d+)*))?\.ya?ml$`)
)

func Scan(basePath string) ([]Service, error) {
	basePath, err := filepath.Abs(basePath)
	if err != nil {
		return nil, err
	}

	services := make([]Service, 0, 80)
	serviceNames := make(map[string]bool)

	// Scan prefixed repos
	for _, prefix := range repoPrefixes {
		dirs, err := filepath.Glob(filepath.Join(basePath, prefix+"*"))
		if err != nil {
			return nil, err
		}

		for _, dir := range dirs {
			info, err := os.Stat(dir)
			if err != nil || !info.IsDir() {
				continue
			}

			// Must have src/main/java to be a service
			if _, err := os.Stat(filepath.Join(dir, "src", "main", "java")); err != nil {
				continue
			}

			name := strings.TrimPrefix(filepath.Base(dir), prefix)
			if serviceNames[name] {
				continue
			}
			serviceNames[name] = true

			integrations := scanRepo(dir)
			matchSpecVersions(dir, integrations)
			svc := Service{
				Name:         name,
				Path:         dir,
				Version:      extractPomVersion(dir),
				Integrations: integrations,
			}
			services = append(services, svc)
		}
	}

	// Scan standalone repos
	for dirName, serviceName := range standaloneRepos {
		dir := filepath.Join(basePath, dirName)
		if _, err := os.Stat(filepath.Join(dir, "src", "main", "java")); err != nil {
			continue
		}
		if serviceNames[serviceName] {
			continue
		}
		serviceNames[serviceName] = true

		integrations := scanRepo(dir)
		matchSpecVersions(dir, integrations)
		svc := Service{
			Name:         serviceName,
			Path:         dir,
			Version:      extractPomVersion(dir),
			Integrations: integrations,
		}
		services = append(services, svc)
	}

	// Normalize CLIENT_IDs: if a CLIENT_ID doesn't resolve to any service but
	// looks like a variant of a known service (e.g. "PartyClient" for "party",
	// "messagingClient" for "messaging"), replace it with the service name.
	nameMap := buildNameMap(serviceNames)
	for i := range services {
		for j := range services[i].Integrations {
			clientID := services[i].Integrations[j].ClientID
			if resolveServiceName(clientID, nameMap) == "" {
				// Try stripping common suffixes like "Client", "Integration"
				lower := strings.ToLower(clientID)
				for _, suffix := range []string{"client", "integration", "service"} {
					trimmed := strings.TrimSuffix(lower, suffix)
					if trimmed != lower {
						if resolved := resolveServiceName(trimmed, nameMap); resolved != "" {
							services[i].Integrations[j].ClientID = resolved
							break
						}
					}
				}
			}
		}
	}

	// Build reverse index

	reverseIndex := make(map[string][]string)
	for i := range services {
		for _, integ := range services[i].Integrations {
			target := resolveServiceName(integ.ClientID, nameMap)
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

	sort.Slice(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})

	return services, nil
}

// buildNameMap creates a lookup from various client ID forms to the canonical service name.
// Handles hyphenation, plurals, and common variations.
func buildNameMap(serviceNames map[string]bool) map[string]string {
	m := make(map[string]string)
	for name := range serviceNames {
		// Exact match
		m[name] = name
		// Hyphen-stripped: "case-data" -> "casedata"
		stripped := strings.ReplaceAll(name, "-", "")
		m[stripped] = name
		// Dot-separated: "case-data" -> "case.data"
		dotted := strings.ReplaceAll(name, "-", ".")
		m[dotted] = name
		// Singular/plural: "relations" <-> "relation"
		if strings.HasSuffix(name, "s") {
			m[strings.TrimSuffix(name, "s")] = name
			m[strings.TrimSuffix(stripped, "s")] = name
		} else {
			m[name+"s"] = name
			m[stripped+"s"] = name
		}
	}
	return m
}

func resolveServiceName(clientID string, nameMap map[string]string) string {
	lower := strings.ToLower(clientID)
	if name, ok := nameMap[lower]; ok {
		return name
	}
	stripped := strings.ReplaceAll(lower, "-", "")
	if name, ok := nameMap[stripped]; ok {
		return name
	}
	return ""
}

func scanRepo(repoPath string) []Integration {
	integrationBase := filepath.Join(repoPath, "src", "main", "java")

	// Track which integration packages we've found, and which have CLIENT_IDs
	allPackages := make(map[string]bool)    // all integration sub-packages seen
	resolvedPkgs := make(map[string]bool)   // packages that had a CLIENT_ID/INTEGRATION_NAME
	seen := make(map[string]bool)           // dedup CLIENT_IDs
	var integrations []Integration

	err := filepath.Walk(integrationBase, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".java") {
			return nil
		}

		rel, _ := filepath.Rel(integrationBase, path)
		if !strings.Contains(rel, "integration") {
			return nil
		}

		// Skip db integration packages
		parts := strings.Split(rel, string(filepath.Separator))
		for _, p := range parts {
			if p == "db" {
				return nil
			}
		}

		// Record the integration sub-package name (skip files directly in integration/)
		pkg := extractPackageName(rel)
		if pkg != "" && !strings.HasSuffix(pkg, ".java") {
			allPackages[pkg] = true
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		content := string(data)

		// Extract CLIENT_ID
		if matches := clientIDRegex.FindStringSubmatch(content); len(matches) > 1 {
			addIntegration(&integrations, seen, matches[1])
			if pkg != "" {
				resolvedPkgs[pkg] = true
			}
		}

		// Extract INTEGRATION_NAME
		if matches := integrationNameRegex.FindStringSubmatch(content); len(matches) > 1 {
			addIntegration(&integrations, seen, matches[1])
			if pkg != "" {
				resolvedPkgs[pkg] = true
			}
		}

		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: error scanning %s: %v\n", repoPath, err)
	}

	// Fallback: integration packages with no CLIENT_ID get the package name as ID
	for pkg := range allPackages {
		if !resolvedPkgs[pkg] && !seen[pkg] {
			seen[pkg] = true
			integrations = append(integrations, Integration{ClientID: pkg})
		}
	}

	sort.Slice(integrations, func(i, j int) bool {
		return integrations[i].ClientID < integrations[j].ClientID
	})

	return integrations
}

// extractPackageName gets the integration sub-package name from the relative file path.
// e.g. "se/sundsvall/foo/integration/casedata/configuration/Config.java" -> "casedata"
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

// extractPomVersion reads the second <version> tag from pom.xml (the project version, not parent).
func extractPomVersion(repoPath string) string {
	data, err := os.ReadFile(filepath.Join(repoPath, "pom.xml"))
	if err != nil {
		return ""
	}
	matches := pomVersionRegex.FindAllStringSubmatch(string(data), 3)
	if len(matches) >= 2 {
		v := strings.TrimSpace(matches[1][1])
		if v != "@project.version@" {
			return v
		}
	}
	return ""
}

// matchSpecVersions finds integration spec files referenced in pom.xml, reads their
// info.version, and matches them to existing integrations.
func matchSpecVersions(repoPath string, integrations []Integration) {
	pomData, err := os.ReadFile(filepath.Join(repoPath, "pom.xml"))
	if err != nil {
		return
	}

	// Find all inputSpec filenames from pom.xml
	specMatches := inputSpecRegex.FindAllStringSubmatch(string(pomData), -1)
	if len(specMatches) == 0 {
		return
	}

	// Build a map: normalized spec filename -> spec version
	specVersions := make(map[string]string)
	for _, m := range specMatches {
		specFilename := m[1]

		// Find the actual file — search common spec directories
		specPath := findSpecFile(repoPath, specFilename)
		if specPath == "" {
			continue
		}

		version := extractSpecVersion(specPath)
		if version == "" {
			continue
		}

		// Normalize the filename to match against CLIENT_IDs
		normalized := normalizeSpecFilename(specFilename)
		specVersions[normalized] = version
	}

	// Match spec versions to integrations
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

// findSpecFile locates a spec file by searching common directories.
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

// extractSpecVersion reads the info.version from a YAML spec file.
// Only matches version lines indented under the info block (indented, before paths:).
func extractSpecVersion(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	// Only search within the first 15 lines (the info block)
	lines := strings.SplitN(string(data), "\n", 16)
	header := strings.Join(lines, "\n")

	if m := specVersionRegex.FindStringSubmatch(header); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

// normalizeSpecFilename strips extension, -api suffix, and version suffixes,
// then lowercases and removes hyphens.
func normalizeSpecFilename(filename string) string {
	// Strip extension and known suffixes
	name := specFileCleanRegex.ReplaceAllString(filename, "")
	// Strip "api-" prefix (e.g. "api-access-mapper.yaml")
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

// FindService returns the service with the given name, or nil.
func FindService(name string, services []Service) *Service {
	for i := range services {
		if services[i].Name == name {
			return &services[i]
		}
	}
	return nil
}

// ResolveTargetName returns the canonical service name for a CLIENT_ID, or "".
func ResolveTargetName(clientID string, services []Service) string {
	nameMap := make(map[string]string)
	for _, s := range services {
		nameMap[s.Name] = s.Name
		nameMap[strings.ReplaceAll(s.Name, "-", "")] = s.Name
		stripped := strings.ReplaceAll(s.Name, "-", "")
		if strings.HasSuffix(s.Name, "s") {
			nameMap[strings.TrimSuffix(s.Name, "s")] = s.Name
			nameMap[strings.TrimSuffix(stripped, "s")] = s.Name
		} else {
			nameMap[s.Name+"s"] = s.Name
			nameMap[stripped+"s"] = s.Name
		}
	}
	return resolveServiceName(clientID, nameMap)
}

// StaleCount returns the number of integrations with a version mismatch.
func StaleCount(svc *Service, services []Service) int {
	count := 0
	for _, integ := range svc.Integrations {
		if integ.SpecVersion == "" {
			continue
		}
		targetName := ResolveTargetName(integ.ClientID, services)
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

// IsInternal checks if a CLIENT_ID maps to a known internal service.
func IsInternal(clientID string, services []Service) bool {
	nameMap := make(map[string]string)
	for _, s := range services {
		nameMap[s.Name] = s.Name
		nameMap[strings.ReplaceAll(s.Name, "-", "")] = s.Name
		// Also add singular/plural
		stripped := strings.ReplaceAll(s.Name, "-", "")
		if strings.HasSuffix(s.Name, "s") {
			nameMap[strings.TrimSuffix(s.Name, "s")] = s.Name
			nameMap[strings.TrimSuffix(stripped, "s")] = s.Name
		} else {
			nameMap[s.Name+"s"] = s.Name
			nameMap[stripped+"s"] = s.Name
		}
	}
	return resolveServiceName(clientID, nameMap) != ""
}
