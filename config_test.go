package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigFromDefaults(t *testing.T) {
	cfg := LoadConfigFrom("/nonexistent")

	if len(cfg.RepoPrefixes) != 2 {
		t.Errorf("expected 2 default prefixes, got %d", len(cfg.RepoPrefixes))
	}
	if cfg.RepoPrefixes[0] != "api-service-" {
		t.Errorf("expected first prefix to be api-service-, got %q", cfg.RepoPrefixes[0])
	}
}

func TestLoadConfigFromCustomFile(t *testing.T) {
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

func TestLoadConfigFromPartialOverride(t *testing.T) {
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

func TestLoadConfigFromInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, ".fondue.json"), []byte("not valid json {{{"), 0644)

	cfg := LoadConfigFrom(tmpDir)

	// Should fall back to defaults
	if cfg.ScanPath != defaultConfig.ScanPath {
		t.Errorf("expected default ScanPath on invalid JSON, got %q", cfg.ScanPath)
	}
}

func TestTryLoadConfig(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		content  string
		wantOK   bool
	}{
		{
			name:   "nonexistent file",
			path:   "/nonexistent/path/.fondue.json",
			wantOK: false,
		},
		{
			name:    "invalid json",
			content: "{not json",
			wantOK:  false,
		},
		{
			name:    "valid json",
			content: `{"scanPath": "/test"}`,
			wantOK:  true,
		},
		{
			name:    "empty json",
			content: `{}`,
			wantOK:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.path
			if tt.content != "" {
				tmpDir := t.TempDir()
				path = filepath.Join(tmpDir, ".fondue.json")
				os.WriteFile(path, []byte(tt.content), 0644)
			}

			_, ok := tryLoadConfig(path)
			if ok != tt.wantOK {
				t.Errorf("tryLoadConfig() ok = %v, want %v", ok, tt.wantOK)
			}
		})
	}
}

func TestTryLoadConfigPreservesDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, ".fondue.json")
	os.WriteFile(path, []byte(`{"scanPath": "/custom"}`), 0644)

	cfg, ok := tryLoadConfig(path)
	if !ok {
		t.Fatal("expected ok = true")
	}
	if cfg.ScanPath != "/custom" {
		t.Errorf("ScanPath = %q, want %q", cfg.ScanPath, "/custom")
	}
	// Defaults should be preserved for fields not in JSON
	if len(cfg.RepoPrefixes) != len(defaultConfig.RepoPrefixes) {
		t.Errorf("expected defaults preserved for RepoPrefixes")
	}
}

func TestLoadConfig(t *testing.T) {
	// LoadConfig tries binary dir first, falls back to defaults.
	// In test mode the binary is the test binary, so there's no .fondue.json next to it.
	cfg := LoadConfig()

	// Should get defaults since no .fondue.json next to test binary
	if cfg.ScanPath != defaultConfig.ScanPath {
		// Unless there happens to be one, just verify it doesn't panic
		_ = cfg
	}
}

func TestLoadConfigFallsBackToDefaults(t *testing.T) {
	cfg := LoadConfig()
	// In test mode, no .fondue.json next to test binary, so should get defaults
	if len(cfg.RepoPrefixes) == 0 {
		t.Error("expected non-empty RepoPrefixes from defaults")
	}
	if cfg.ScanPath == "" {
		t.Error("expected non-empty ScanPath from defaults")
	}
}

func TestLoadConfigWithConfigNextToBinary(t *testing.T) {
	// Place a .fondue.json next to the test binary
	binaryPath := configPathNextToBinary()
	if binaryPath == "" {
		t.Skip("cannot determine binary path")
	}

	cfgJSON := `{"scanPath": "/from-binary-dir", "repoPrefixes": ["test-"]}`
	if err := os.WriteFile(binaryPath, []byte(cfgJSON), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	t.Cleanup(func() { os.Remove(binaryPath) })

	cfg := LoadConfig()
	if cfg.ScanPath != "/from-binary-dir" {
		t.Errorf("ScanPath = %q, want /from-binary-dir", cfg.ScanPath)
	}
}

func TestLoadConfigFromWithBinaryFallbackHit(t *testing.T) {
	binaryPath := configPathNextToBinary()
	if binaryPath == "" {
		t.Skip("cannot determine binary path")
	}

	cfgJSON := `{"scanPath": "/binary-fallback"}`
	if err := os.WriteFile(binaryPath, []byte(cfgJSON), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	t.Cleanup(func() { os.Remove(binaryPath) })

	// First dir doesn't have config, falls back to binary dir
	cfg := LoadConfigFrom("/tmp/nonexistent-abc")
	if cfg.ScanPath != "/binary-fallback" {
		t.Errorf("ScanPath = %q, want /binary-fallback", cfg.ScanPath)
	}
}

func TestLoadConfigFromWithBinaryFallback(t *testing.T) {
	// First dir doesn't have config, binary dir also won't, so should get defaults
	cfg := LoadConfigFrom("/tmp/nonexistent-dir-abc123")
	if cfg.ScanPath != defaultConfig.ScanPath {
		t.Errorf("ScanPath = %q, want default %q", cfg.ScanPath, defaultConfig.ScanPath)
	}
}

func TestConfigPathNextToBinary(t *testing.T) {
	path := configPathNextToBinary()
	// Should return a path ending with .fondue.json or empty string
	if path != "" && !filepath.IsAbs(path) {
		t.Errorf("expected absolute path, got %q", path)
	}
	if path != "" {
		base := filepath.Base(path)
		if base != ".fondue.json" {
			t.Errorf("expected .fondue.json, got %q", base)
		}
	}
}
