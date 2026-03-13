package graph

import (
	"fmt"
	"sort"
	"strings"
)

// DiffEntry describes a change in the service graph.
type DiffEntry struct {
	Kind    DiffKind
	Service string
	Detail  string
}

// DiffKind classifies a diff entry.
type DiffKind int

const (
	DiffAdded          DiffKind = iota // Service added
	DiffRemoved                        // Service removed
	DiffDepAdded                       // Dependency added to existing service
	DiffDepRemoved                     // Dependency removed from existing service
	DiffVersionChanged                 // Service version changed
)

// DiffService is a simplified service representation for diffing JSON exports.
type DiffService struct {
	Name         string   `json:"name"`
	Version      string   `json:"version,omitempty"`
	Dependencies []string `json:"dependencies"`
	DependedOnBy []string `json:"dependedOnBy"`
}

// ComputeDiff compares two snapshots and returns a list of changes.
func ComputeDiff(old, new []DiffService) []DiffEntry {
	oldMap := make(map[string]*DiffService)
	for i := range old {
		oldMap[old[i].Name] = &old[i]
	}
	newMap := make(map[string]*DiffService)
	for i := range new {
		newMap[new[i].Name] = &new[i]
	}

	var entries []DiffEntry

	// Find removed services
	for _, svc := range old {
		if _, exists := newMap[svc.Name]; !exists {
			entries = append(entries, DiffEntry{Kind: DiffRemoved, Service: svc.Name})
		}
	}

	// Find added services and changes
	for _, svc := range new {
		oldSvc, exists := oldMap[svc.Name]
		if !exists {
			entries = append(entries, DiffEntry{Kind: DiffAdded, Service: svc.Name})
			continue
		}

		// Version change
		if oldSvc.Version != svc.Version && (oldSvc.Version != "" || svc.Version != "") {
			entries = append(entries, DiffEntry{
				Kind:    DiffVersionChanged,
				Service: svc.Name,
				Detail:  fmt.Sprintf("%s → %s", versionStr(oldSvc.Version), versionStr(svc.Version)),
			})
		}

		// Dependency changes
		oldDeps := toSet(oldSvc.Dependencies)
		newDeps := toSet(svc.Dependencies)

		for dep := range newDeps {
			if !oldDeps[dep] {
				entries = append(entries, DiffEntry{Kind: DiffDepAdded, Service: svc.Name, Detail: dep})
			}
		}
		for dep := range oldDeps {
			if !newDeps[dep] {
				entries = append(entries, DiffEntry{Kind: DiffDepRemoved, Service: svc.Name, Detail: dep})
			}
		}
	}

	// Sort: removed first, then added, then changes
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Kind != entries[j].Kind {
			return entries[i].Kind < entries[j].Kind
		}
		return entries[i].Service < entries[j].Service
	})

	return entries
}

// FormatDiff returns a human-readable string of the diff.
func FormatDiff(entries []DiffEntry) string {
	if len(entries) == 0 {
		return "No changes detected."
	}

	var s strings.Builder
	s.WriteString(fmt.Sprintf("%d change(s) detected:\n\n", len(entries)))

	for _, e := range entries {
		switch e.Kind {
		case DiffAdded:
			s.WriteString(fmt.Sprintf("  + %s  (new service)\n", e.Service))
		case DiffRemoved:
			s.WriteString(fmt.Sprintf("  - %s  (removed)\n", e.Service))
		case DiffDepAdded:
			s.WriteString(fmt.Sprintf("  ~ %s  +dep %s\n", e.Service, e.Detail))
		case DiffDepRemoved:
			s.WriteString(fmt.Sprintf("  ~ %s  -dep %s\n", e.Service, e.Detail))
		case DiffVersionChanged:
			s.WriteString(fmt.Sprintf("  ~ %s  version %s\n", e.Service, e.Detail))
		}
	}

	return s.String()
}

func toSet(items []string) map[string]bool {
	s := make(map[string]bool)
	for _, item := range items {
		s[item] = true
	}
	return s
}

func versionStr(v string) string {
	if v == "" {
		return "(none)"
	}
	return v
}
