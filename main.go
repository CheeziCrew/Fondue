package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cheezi/service-map/scanner"
	"github.com/cheezi/service-map/tui"
)

func main() {
	scanPath := "~/code/scit/"
	listMode := false

	args := os.Args[1:]
	for _, arg := range args {
		if arg == "--list" || arg == "-l" {
			listMode = true
		} else {
			scanPath = arg
		}
	}

	// Expand ~
	if scanPath[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		scanPath = filepath.Join(home, scanPath[2:])
	}

	fmt.Fprintf(os.Stderr, "Scanning %s ...\n", scanPath)

	services, err := scanner.Scan(scanPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error scanning: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Found %d services\n", len(services))

	if listMode {
		for _, svc := range services {
			ver := svc.Version
			if ver == "" {
				ver = "?"
			}
			fmt.Printf("%-30s v%-6s  deps: %-3d  dependents: %-3d", svc.Name, ver, len(svc.Integrations), len(svc.DependedOnBy))
			stale := scanner.StaleCount(&svc, services)
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
		return
	}

	p := tea.NewProgram(tui.NewModel(services), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
