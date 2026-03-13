package graph

import (
	"fmt"
	"strings"

	"github.com/CheeziCrew/fondue/scanner"
)

// ImpactEntry represents a service affected by a change, with its BFS depth.
type ImpactEntry struct {
	Name  string
	Depth int
}

// ImpactAnalysis performs a reverse BFS from the given root service,
// following DependedOnBy edges to find all transitively affected downstream consumers.
func ImpactAnalysis(root string, services []scanner.Service, idx *scanner.NameIndex) []ImpactEntry {
	byName := buildServiceIndex(services)
	visited := make(map[string]bool)
	visited[root] = true

	type queueItem struct {
		name  string
		depth int
	}
	queue := []queueItem{{root, 0}}
	var result []ImpactEntry

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		svc := byName[cur.name]
		if svc == nil {
			continue
		}

		for _, dep := range svc.DependedOnBy {
			if !visited[dep] {
				visited[dep] = true
				entry := ImpactEntry{Name: dep, Depth: cur.depth + 1}
				result = append(result, entry)
				queue = append(queue, queueItem{dep, cur.depth + 1})
			}
		}
	}

	return result
}

// FormatImpact returns a human-readable string of the impact analysis results.
func FormatImpact(root string, entries []ImpactEntry) string {
	if len(entries) == 0 {
		return fmt.Sprintf("No downstream consumers found for %s.", root)
	}

	var s strings.Builder
	s.WriteString(fmt.Sprintf("%s affects %d service(s):\n\n", root, len(entries)))

	prevDepth := 0
	for _, e := range entries {
		if e.Depth != prevDepth {
			if prevDepth > 0 {
				s.WriteString("\n")
			}
			s.WriteString(fmt.Sprintf("  Depth %d:\n", e.Depth))
			prevDepth = e.Depth
		}
		s.WriteString(fmt.Sprintf("    %s\n", e.Name))
	}
	return s.String()
}
