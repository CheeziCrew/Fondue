package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/CheeziCrew/fondue/graph"
	"github.com/CheeziCrew/fondue/scanner"
	"github.com/CheeziCrew/fondue/tui"
)

var version = "dev"

// exitFunc allows tests to intercept os.Exit calls.
var exitFunc = os.Exit

type outputMode int

const (
	modeInteractive outputMode = iota
	modeList
	modeJSON
	modeDOT
	modeCycles
	modeImpact
	modeDiff
)

func main() {
	run(os.Args[1:])
}

type parsedArgs struct {
	mode         outputMode
	customPath   string
	impactTarget string
	diffFiles    [2]string
}

func parseArgs(args []string) parsedArgs {
	var p parsedArgs
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--list", "-l":
			p.mode = modeList
		case "--json", "-j":
			p.mode = modeJSON
		case "--dot", "-d":
			p.mode = modeDOT
		case "--cycles", "-c":
			p.mode = modeCycles
		case "--impact":
			p.mode = modeImpact
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				p.impactTarget = args[i]
			}
		case "--diff":
			p.mode = modeDiff
			for n := 0; n < 2 && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-"); n++ {
				i++
				p.diffFiles[n] = args[i]
			}
		case "--help", "-h":
			p.mode = -1 // sentinel for help
		case "--version", "-v":
			p.mode = -2 // sentinel for version
		default:
			p.customPath = args[i]
		}
	}
	return p
}

func run(args []string) {
	p := parseArgs(args)

	switch p.mode {
	case -1:
		printUsage()
		return
	case -2:
		fmt.Printf("fondue %s\n", version)
		return
	}

	// Diff mode doesn't need scanning — handle early.
	if p.mode == modeDiff {
		printDiff(p.diffFiles[0], p.diffFiles[1])
		return
	}

	cfg := LoadConfig()
	scanPath := cfg.ScanPath

	// CLI path overrides config
	if p.customPath != "" {
		scanPath = p.customPath
	}

	// Expand ~
	if len(scanPath) >= 2 && scanPath[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			exitFunc(1)
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
		exitFunc(1)
	}

	fmt.Fprintf(os.Stderr, "Found %d services\n", len(services))

	switch p.mode {
	case modeList:
		printList(services, idx)
	case modeJSON:
		printJSON(services)
	case modeDOT:
		WriteDOT(os.Stdout, services, idx, "")
	case modeCycles:
		printCycles(services, idx)
	case modeImpact:
		printImpact(p.impactTarget, services, idx)
	default:
		prog := tea.NewProgram(tui.NewModel(services, idx))
		if _, err := prog.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			exitFunc(1)
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
  -c, --cycles     Detect and print dependency cycles
      --impact SVC Show transitively affected downstream consumers
      --diff A B   Compare two JSON exports and show changes
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

func printDiff(oldPath, newPath string) {
	if oldPath == "" || newPath == "" {
		fmt.Fprintln(os.Stderr, "Usage: fondue --diff <old.json> <new.json>")
		exitFunc(1)
	}

	oldServices, err := loadDiffJSON(oldPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading %s: %v\n", oldPath, err)
		exitFunc(1)
	}

	newServices, err := loadDiffJSON(newPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading %s: %v\n", newPath, err)
		exitFunc(1)
	}

	entries := graph.ComputeDiff(oldServices, newServices)
	fmt.Println(graph.FormatDiff(entries))
}

func loadDiffJSON(path string) ([]graph.DiffService, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var services []graph.DiffService
	if err := json.Unmarshal(data, &services); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return services, nil
}

func printImpact(target string, services []scanner.Service, idx *scanner.NameIndex) {
	if target == "" {
		fmt.Fprintln(os.Stderr, "Usage: fondue --impact <service-name>")
		exitFunc(1)
	}

	// Try to resolve the target name
	resolved := idx.Resolve(target)
	if resolved == "" {
		resolved = target
	}

	entries := graph.ImpactAnalysis(resolved, services, idx)
	fmt.Println(graph.FormatImpact(resolved, entries))
}

func printCycles(services []scanner.Service, idx *scanner.NameIndex) {
	cycles := graph.DetectCycles(services, idx)
	if len(cycles) == 0 {
		fmt.Println("No dependency cycles detected.")
		return
	}
	fmt.Printf("Found %d dependency cycle(s):\n\n", len(cycles))
	fmt.Println(graph.FormatCycles(cycles))
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
