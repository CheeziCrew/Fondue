package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/CheeziCrew/fondue/scanner"
	"github.com/CheeziCrew/fondue/tui"
)

var version = "dev"

type outputMode int

const (
	modeInteractive outputMode = iota
	modeList
	modeJSON
	modeDOT
)

func main() {
	mode := modeInteractive
	var customPath string

	args := os.Args[1:]
	for _, arg := range args {
		switch arg {
		case "--list", "-l":
			mode = modeList
		case "--json", "-j":
			mode = modeJSON
		case "--dot", "-d":
			mode = modeDOT
		case "--help", "-h":
			printUsage()
			return
		case "--version", "-v":
			fmt.Printf("fondue %s\n", version)
			return
		default:
			customPath = arg
		}
	}

	cfg := LoadConfig()
	scanPath := cfg.ScanPath

	// CLI path overrides config
	if customPath != "" {
		scanPath = customPath
	}

	// Expand ~
	if len(scanPath) >= 2 && scanPath[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		scanPath = filepath.Join(home, scanPath[2:])
	}

	// Reload config from scan dir (may have .fondue.json there)
	cfg = LoadConfigFrom(scanPath)

	fmt.Fprintf(os.Stderr, "Scanning %s ...\n", scanPath)

	scanCfg := scanner.ScanConfig{
		RepoPrefixes:    cfg.RepoPrefixes,
		StandaloneRepos: cfg.StandaloneRepos,
	}

	services, idx, err := scanner.Scan(scanPath, scanCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error scanning: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Found %d services\n", len(services))

	switch mode {
	case modeList:
		printList(services, idx)
	case modeJSON:
		printJSON(services)
	case modeDOT:
		printDOT(services, idx)
	default:
		p := tea.NewProgram(tui.NewModel(services, idx), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `fondue %s — melt through your microservice dependencies

Usage:
  fondue [path] [flags]

Flags:
  -l, --list       Print services as a table
  -j, --json       Export dependency graph as JSON
  -d, --dot        Export as Graphviz DOT format
  -v, --version    Print version
  -h, --help       Show this help

Config:
  Drop .fondue.json in your scan directory or next to the binary.

  {
    "scanPath": "~/code/scit/",
    "repoPrefixes": ["api-service-", "pw-"],
    "standaloneRepos": {"api-comfact-facade": "comfact-facade"}
  }

Examples:
  fondue                            Interactive TUI (default path from config)
  fondue ~/code/myproject           Scan a specific directory
  fondue --json | jq                Pipe JSON to jq
  fondue --dot | dot -Tpng          Generate dependency graph image
`, version)
}

func printList(services []scanner.Service, idx *scanner.NameIndex) {
	for _, svc := range services {
		ver := svc.Version
		if ver == "" {
			ver = "?"
		}
		fmt.Printf("%-30s v%-6s  deps: %-3d  dependents: %-3d", svc.Name, ver, len(svc.Integrations), len(svc.DependedOnBy))
		stale := scanner.StaleCount(&svc, services, idx)
		if stale > 0 {
			fmt.Printf("  ⚠ %d stale", stale)
		}
		if len(svc.Integrations) > 0 {
			var ids []string
			for _, i := range svc.Integrations {
				ids = append(ids, i.ClientID)
			}
			fmt.Printf("  -> %s", strings.Join(ids, ", "))
		}
		fmt.Println()
	}
}

// ── JSON export ─────────────────────────────────────────────────────

type jsonService struct {
	Name         string   `json:"name"`
	Version      string   `json:"version,omitempty"`
	Path         string   `json:"path"`
	Dependencies []string `json:"dependencies"`
	DependedOnBy []string `json:"dependedOnBy"`
}

func printJSON(services []scanner.Service) {
	out := make([]jsonService, len(services))
	for i, svc := range services {
		var deps []string
		for _, integ := range svc.Integrations {
			deps = append(deps, integ.ClientID)
		}
		out[i] = jsonService{
			Name:         svc.Name,
			Version:      svc.Version,
			Path:         svc.Path,
			Dependencies: deps,
			DependedOnBy: svc.DependedOnBy,
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(out)
}

// ── DOT (Graphviz) export ───────────────────────────────────────────

// heatColor maps a 0.0–1.0 intensity to a cool→hot gradient (blue→cyan→green→yellow→red).
func heatColor(t float64) string {
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

func printDOT(services []scanner.Service, idx *scanner.NameIndex) {
	// Find max dependents for heat scaling
	maxDeps := 0
	for _, svc := range services {
		total := len(svc.DependedOnBy) + len(svc.Integrations)
		if total > maxDeps {
			maxDeps = total
		}
	}

	fmt.Println("digraph services {")
	fmt.Println("  bgcolor=\"#1a1b26\";")
	fmt.Println("  rankdir=LR;")
	fmt.Println("  pad=0.5;")
	fmt.Println("  nodesep=0.6;")
	fmt.Println("  ranksep=1.2;")
	fmt.Println("  concentrate=false;")
	fmt.Println("  splines=true;")
	fmt.Println("  node [shape=box, style=\"rounded,filled\", fontname=\"Helvetica\", fontsize=11, fontcolor=\"#c0caf5\", color=\"#3b4261\", fillcolor=\"#24283b\", penwidth=1.5];")
	fmt.Println("  edge [color=\"#3b4261\", arrowsize=0.7, penwidth=1.2];")
	fmt.Println()

	for _, svc := range services {
		label := svc.Name
		if svc.Version != "" {
			label += "\\n" + svc.Version
		}
		total := len(svc.DependedOnBy) + len(svc.Integrations)
		t := 0.0
		if maxDeps > 0 {
			t = float64(total) / float64(maxDeps)
		}
		color := heatColor(t)
		stale := scanner.StaleCount(&svc, services, idx)
		fill := "#24283b"
		if stale > 0 {
			fill = "#2d1f2f"
			label += fmt.Sprintf("\\n⚠ %d stale", stale)
		}
		fmt.Printf("  \"%s\" [label=\"%s\", color=\"%s\", fillcolor=\"%s\", penwidth=%.1f];\n", svc.Name, label, color, fill, 1.5+t*2.5)
	}

	fmt.Println()

	externals := make(map[string]bool)

	for _, svc := range services {
		total := len(svc.DependedOnBy) + len(svc.Integrations)
		t := 0.0
		if maxDeps > 0 {
			t = float64(total) / float64(maxDeps)
		}
		edgeColor := heatColor(t * 0.6)
		for _, integ := range svc.Integrations {
			target := idx.Resolve(integ.ClientID)
			if target != "" {
				fmt.Printf("  \"%s\" -> \"%s\" [color=\"%s\"];\n", svc.Name, target, edgeColor)
			} else {
				extID := strings.ToLower(integ.ClientID)
				externals[extID] = true
				fmt.Printf("  \"%s\" -> \"%s\" [color=\"%s\", style=dashed];\n", svc.Name, extID, edgeColor)
			}
		}
	}

	if len(externals) > 0 {
		fmt.Println()
		fmt.Println("  subgraph cluster_external {")
		fmt.Println("    style=dashed; color=\"#3b4261\"; fontcolor=\"#565f89\"; fontname=\"Helvetica\"; label=\"external\";")
		for ext := range externals {
			fmt.Printf("    \"%s\" [style=\"rounded,dashed,filled\", fillcolor=\"#1a1b26\", color=\"#565f89\", fontcolor=\"#565f89\"];\n", ext)
		}
		fmt.Println("  }")
	}

	fmt.Println("}")
}
