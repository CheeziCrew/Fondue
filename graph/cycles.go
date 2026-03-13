package graph

import "github.com/CheeziCrew/fondue/scanner"

// Cycle represents a detected cycle as an ordered list of service names.
// The last element connects back to the first.
type Cycle []string

// DetectCycles finds all unique cycles in the service dependency graph
// using DFS with back-edge detection.
func DetectCycles(services []scanner.Service, idx *scanner.NameIndex) []Cycle {
	byName := buildServiceIndex(services)
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	var cycles []Cycle
	seen := make(map[string]bool) // dedup cycles by canonical form

	for _, svc := range services {
		if !visited[svc.Name] {
			var path []string
			dfs(svc.Name, byName, idx, visited, recStack, path, &cycles, seen)
		}
	}
	return cycles
}

func dfs(node string, byName map[string]*scanner.Service, idx *scanner.NameIndex,
	visited, recStack map[string]bool, path []string, cycles *[]Cycle, seen map[string]bool) {

	visited[node] = true
	recStack[node] = true
	path = append(path, node)

	svc := byName[node]
	if svc != nil {
		for _, integ := range svc.Integrations {
			target := idx.Resolve(integ.ClientID)
			if target == "" {
				continue
			}
			if recStack[target] {
				// Found a cycle — extract it from path.
				cycle := extractCycle(path, target)
				if cycle != nil {
					key := canonicalKey(cycle)
					if !seen[key] {
						seen[key] = true
						*cycles = append(*cycles, cycle)
					}
				}
			} else if !visited[target] {
				dfs(target, byName, idx, visited, recStack, path, cycles, seen)
			}
		}
	}

	recStack[node] = false
}

// extractCycle returns the cycle from start to the end of path.
func extractCycle(path []string, target string) Cycle {
	for i, name := range path {
		if name == target {
			cycle := make(Cycle, len(path)-i)
			copy(cycle, path[i:])
			return cycle
		}
	}
	return nil
}

// canonicalKey produces a dedup key for a cycle by starting at the smallest element.
func canonicalKey(cycle Cycle) string {
	if len(cycle) == 0 {
		return ""
	}
	minIdx := 0
	for i, name := range cycle {
		if name < cycle[minIdx] {
			minIdx = i
		}
	}
	key := ""
	for i := 0; i < len(cycle); i++ {
		if i > 0 {
			key += " → "
		}
		key += cycle[(minIdx+i)%len(cycle)]
	}
	return key
}

// FormatCycles returns a human-readable string of detected cycles.
func FormatCycles(cycles []Cycle) string {
	if len(cycles) == 0 {
		return "No cycles detected."
	}
	var s string
	for i, cycle := range cycles {
		s += cycleString(cycle)
		if i < len(cycles)-1 {
			s += "\n"
		}
	}
	return s
}

func cycleString(cycle Cycle) string {
	s := ""
	for i, name := range cycle {
		if i > 0 {
			s += " → "
		}
		s += name
	}
	s += " → " + cycle[0] // close the cycle
	return s
}
