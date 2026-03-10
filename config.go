package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds user-configurable settings for fondue.
// Loaded from .fondue.json next to the binary or in the scan directory.
type Config struct {
	ScanPath       string            `json:"scanPath"`
	RepoPrefixes   []string          `json:"repoPrefixes"`
	StandaloneRepos map[string]string `json:"standaloneRepos"`
}

var defaultConfig = Config{
	ScanPath:     "~/code/scit/",
	RepoPrefixes: []string{"api-service-", "pw-"},
	StandaloneRepos: map[string]string{
		"api-comfact-facade": "comfact-facade",
		"cimd-proxy":         "cimd-proxy",
		"formpipe-proxy":     "formpipe-proxy",
	},
}

// LoadConfig tries to load .fondue.json from next to the binary, falls back to defaults.
func LoadConfig() Config {
	if cfg, ok := tryLoadConfig(configPathNextToBinary()); ok {
		return cfg
	}
	return defaultConfig
}

// LoadConfigFrom tries a specific directory first, then binary dir, then defaults.
func LoadConfigFrom(dir string) Config {
	if cfg, ok := tryLoadConfig(filepath.Join(dir, ".fondue.json")); ok {
		return cfg
	}
	if cfg, ok := tryLoadConfig(configPathNextToBinary()); ok {
		return cfg
	}
	return defaultConfig
}

func tryLoadConfig(path string) (Config, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, false
	}

	cfg := defaultConfig // start with defaults so missing fields keep defaults
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, false
	}
	return cfg, true
}

func configPathNextToBinary() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Join(filepath.Dir(exe), ".fondue.json")
}
