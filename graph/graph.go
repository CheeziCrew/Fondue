// Package graph provides DOT generation and subgraph extraction for fondue.
package graph

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/CheeziCrew/fondue/scanner"
)

// HeatColor maps a 0.0–1.0 intensity to a cool→hot gradient (blue→cyan→green→yellow→red).
func HeatColor(t float64) string {
	type rgb struct{ r, g, b int }
	stops := []rgb{{66, 133, 244}, {52, 211, 153}, {251, 191, 36}, {239, 68, 68}}
	t = max(0, min(1, t))
	scaled := t * float64(len(stops)-1)
	i := int(scaled)
	if i >= len(stops)-1 {
		return fmt.Sprintf("#%02x%02x%02x", stops[len(stops)-1].r, stops[len(stops)-1].g, stops[len(stops)-1].b)
	}
	f := scaled - float64(i)
	r := int(float64(stops[i].r)*(1-f) + float64(stops[i+1].r)*f)
	g := int(float64(stops[i].g)*(1-f) + float64(stops[i+1].g)*f)
	b := int(float64(stops[i].b)*(1-f) + float64(stops[i+1].b)*f)
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

func maxDepCount(services []scanner.Service) int {
	m := 0
	for _, svc := range services {
		if total := len(svc.DependedOnBy) + len(svc.Integrations); total > m {
			m = total
		}
	}
	return m
}

func heatIntensity(svc scanner.Service, maxDeps int) float64 {
	if maxDeps == 0 {
		return 0
	}
	total := len(svc.DependedOnBy) + len(svc.Integrations)
	return float64(total) / float64(maxDeps)
}

func writeDOTHeader(w io.Writer) {
	fmt.Fprintln(w, "digraph services {")
	fmt.Fprintln(w, "  bgcolor=\"#1a1b26\";")
	fmt.Fprintln(w, "  rankdir=LR;")
	fmt.Fprintln(w, "  pad=0.5;")
	fmt.Fprintln(w, "  nodesep=0.6;")
	fmt.Fprintln(w, "  ranksep=1.2;")
	fmt.Fprintln(w, "  concentrate=false;")
	fmt.Fprintln(w, "  splines=true;")
	fmt.Fprintln(w, "  node [shape=box, style=\"rounded,filled\", fontname=\"Helvetica\", fontsize=11, fontcolor=\"#c0caf5\", color=\"#3b4261\", fillcolor=\"#24283b\", penwidth=1.5];")
	fmt.Fprintln(w, "  edge [color=\"#3b4261\", arrowsize=0.7, penwidth=1.2];")
	fmt.Fprintln(w)
}

func writeDOTNodes(w io.Writer, services []scanner.Service, idx *scanner.NameIndex, maxDeps int, highlight string) {
	for _, svc := range services {
		label := buildNodeLabel(svc, services, idx)
		t := heatIntensity(svc, maxDeps)
		color := HeatColor(t)
		fill := "#24283b"
		if scanner.StaleCount(&svc, services, idx) > 0 {
			fill = "#2d1f2f"
		}
		pw := 1.5 + t*2.5
		if highlight != "" && svc.Name == highlight {
			color = "#ff9e64"
			pw = 4.0
		}
		fmt.Fprintf(w, "  \"%s\" [label=\"%s\", color=\"%s\", fillcolor=\"%s\", penwidth=%.1f];\n", svc.Name, label, color, fill, pw)
	}
}

func buildNodeLabel(svc scanner.Service, services []scanner.Service, idx *scanner.NameIndex) string {
	label := svc.Name
	if svc.Version != "" {
		label += "\\n" + svc.Version
	}
	if stale := scanner.StaleCount(&svc, services, idx); stale > 0 {
		label += fmt.Sprintf("\\n⚠ %d stale", stale)
	}
	return label
}

func writeDOTEdges(w io.Writer, services []scanner.Service, idx *scanner.NameIndex, maxDeps int) map[string]bool {
	fmt.Fprintln(w)
	externals := make(map[string]bool)
	for _, svc := range services {
		t := heatIntensity(svc, maxDeps)
		edgeColor := HeatColor(t * 0.6)
		writeServiceEdges(w, svc, idx, edgeColor, externals)
	}
	return externals
}

func writeServiceEdges(w io.Writer, svc scanner.Service, idx *scanner.NameIndex, edgeColor string, externals map[string]bool) {
	for _, integ := range svc.Integrations {
		target := idx.Resolve(integ.ClientID)
		if target != "" {
			fmt.Fprintf(w, "  \"%s\" -> \"%s\" [color=\"%s\"];\n", svc.Name, target, edgeColor)
		} else {
			extID := strings.ToLower(integ.ClientID)
			externals[extID] = true
			fmt.Fprintf(w, "  \"%s\" -> \"%s\" [color=\"%s\", style=dashed];\n", svc.Name, extID, edgeColor)
		}
	}
}

func writeDOTExternals(w io.Writer, externals map[string]bool) {
	if len(externals) == 0 {
		return
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  subgraph cluster_external {")
	fmt.Fprintln(w, "    style=dashed; color=\"#3b4261\"; fontcolor=\"#565f89\"; fontname=\"Helvetica\"; label=\"external\";")
	for ext := range externals {
		fmt.Fprintf(w, "    \"%s\" [style=\"rounded,dashed,filled\", fillcolor=\"#1a1b26\", color=\"#565f89\", fontcolor=\"#565f89\"];\n", ext)
	}
	fmt.Fprintln(w, "  }")
}

// WriteDOT writes a Graphviz DOT representation of the service graph to w.
// If highlight is non-empty, that service node gets a distinct border.
func WriteDOT(w io.Writer, services []scanner.Service, idx *scanner.NameIndex, highlight string) {
	maxDeps := maxDepCount(services)
	writeDOTHeader(w)
	writeDOTNodes(w, services, idx, maxDeps, highlight)
	externals := writeDOTEdges(w, services, idx, maxDeps)
	writeDOTExternals(w, externals)
	fmt.Fprintln(w, "}")
}

type bfsEntry struct {
	name  string
	depth int
}

func buildServiceIndex(services []scanner.Service) map[string]*scanner.Service {
	byName := make(map[string]*scanner.Service)
	for i := range services {
		byName[services[i].Name] = &services[i]
	}
	return byName
}

func enqueueNeighbors(svc *scanner.Service, idx *scanner.NameIndex, depth int, visited map[string]bool) []bfsEntry {
	var neighbors []bfsEntry
	for _, integ := range svc.Integrations {
		if target := idx.Resolve(integ.ClientID); target != "" && !visited[target] {
			visited[target] = true
			neighbors = append(neighbors, bfsEntry{target, depth})
		}
	}
	for _, dep := range svc.DependedOnBy {
		if !visited[dep] {
			visited[dep] = true
			neighbors = append(neighbors, bfsEntry{dep, depth})
		}
	}
	return neighbors
}

// CollectSubgraph does a BFS from root, collecting all services within N hops
// in both directions (outbound integrations + inbound dependents).
func CollectSubgraph(root string, services []scanner.Service, idx *scanner.NameIndex, hops int) []scanner.Service {
	visited := make(map[string]bool)
	visited[root] = true
	queue := []bfsEntry{{root, 0}}
	byName := buildServiceIndex(services)

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		if cur.depth >= hops {
			continue
		}

		svc := byName[cur.name]
		if svc == nil {
			continue
		}

		queue = append(queue, enqueueNeighbors(svc, idx, cur.depth+1, visited)...)
	}

	var result []scanner.Service
	for _, svc := range services {
		if visited[svc.Name] {
			result = append(result, svc)
		}
	}
	return result
}

// ExportPNG generates a subgraph DOT, pipes through graphviz, writes a temp PNG, and opens it.
// Returns the temp file path or an error.
func ExportPNG(root string, services []scanner.Service, idx *scanner.NameIndex, hops int) (string, error) {
	if _, err := lookPath("dot"); err != nil {
		return "", fmt.Errorf("graphviz not installed (brew install graphviz)")
	}

	subset := CollectSubgraph(root, services, idx, hops)

	var buf bytes.Buffer
	WriteDOT(&buf, subset, idx, root)

	tmpFile, err := os.CreateTemp("", "fondue-*.png")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	tmpFile.Close()

	if err := runDot(buf.Bytes(), tmpFile.Name()); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("dot: %s", err.Error())
	}

	if err := openFile(tmpFile.Name()); err != nil {
		return tmpFile.Name(), fmt.Errorf("open: %w", err)
	}

	return tmpFile.Name(), nil
}
