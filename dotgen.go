package main

import (
	"io"

	"github.com/CheeziCrew/fondue/graph"
	"github.com/CheeziCrew/fondue/scanner"
)

// WriteDOT delegates to the graph package for DOT generation.
func WriteDOT(w io.Writer, services []scanner.Service, idx *scanner.NameIndex, highlight string) {
	graph.WriteDOT(w, services, idx, highlight)
}
