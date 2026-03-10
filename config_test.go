package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigFrom_Defaults(t *testing.T) {
	cfg := LoadConfigFrom("/nonexistent")

	if len(cfg.RepoPrefixes) != 2 {
		t.Errorf("expected 2 default prefixes, got %d", len(cfg.RepoPrefixes))
	}
	if cfg.RepoPrefixes[0] != "api-service-" {
		t.Errorf("expected first prefix to be api-service-, got %q", cfg.RepoPrefixes[0])
	}
}

func TestLoadConfigFrom_CustomFile(t *testing.T) {
	tmpDir := t.TempDir()

	cfgJSON := `{
  "scanPath": "~/custom/path",
  "repoPrefixes": ["my-svc-", "backend-"],
  "standaloneRepos": {"my-proxy": "proxy"}
}`
	os.WriteFile(filepath.Join(tmpDir, ".fondue.json"), []byte(cfgJSON), 0644)

	cfg := LoadConfigFrom(tmpDir)

	if cfg.ScanPath != "~/custom/path" {
		t.Errorf("ScanPath = %q, want %q", cfg.ScanPath, "~/custom/path")
	}
	if len(cfg.RepoPrefixes) != 2 {
		t.Errorf("expected 2 prefixes, got %d", len(cfg.RepoPrefixes))
	}
	if cfg.RepoPrefixes[0] != "my-svc-" {
		t.Errorf("prefix[0] = %q, want %q", cfg.RepoPrefixes[0], "my-svc-")
	}
	if cfg.StandaloneRepos["my-proxy"] != "proxy" {
		t.Errorf("standalone repo mismatch")
	}
}

func TestLoadConfigFrom_PartialOverride(t *testing.T) {
	tmpDir := t.TempDir()

	// Only override prefixes, keep defaults for the rest
	cfgJSON := `{"repoPrefixes": ["custom-"]}`
	os.WriteFile(filepath.Join(tmpDir, ".fondue.json"), []byte(cfgJSON), 0644)

	cfg := LoadConfigFrom(tmpDir)

	if len(cfg.RepoPrefixes) != 1 {
		t.Errorf("expected 1 prefix, got %d", len(cfg.RepoPrefixes))
	}
	// StandaloneRepos should keep defaults
	if len(cfg.StandaloneRepos) == 0 {
		t.Error("expected default standalone repos to be preserved")
	}
}
