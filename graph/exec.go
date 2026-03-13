package graph

import (
	"os/exec"
	"strings"
)

// Package-level function variables for testability.
var lookPath = exec.LookPath

var runDot = func(input []byte, outPath string) error {
	cmd := exec.Command("dot", "-Tpng", "-o", outPath)
	cmd.Stdin = strings.NewReader(string(input))
	if out, err := cmd.CombinedOutput(); err != nil {
		return &dotError{msg: strings.TrimSpace(string(out))}
	}
	return nil
}

var openFile = func(path string) error {
	cmd := exec.Command("open", path)
	return cmd.Start()
}

type dotError struct {
	msg string
}

func (e *dotError) Error() string {
	return e.msg
}
