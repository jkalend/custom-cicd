package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg == nil || cfg.APIURL == "" {
		t.Fatalf("expected non-empty default config, got %+v", cfg)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("USERPROFILE")
	if origHome == "" {
		origHome = os.Getenv("HOME")
	}
	t.Setenv("USERPROFILE", tmpDir)
	t.Setenv("HOME", tmpDir)

	cfg := &Config{APIURL: "http://127.0.0.1:8888"}

	// Save first time
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Save second time (should NOT fail with 'file already exists')
	cfg.APIURL = "http://127.0.0.1:9999"
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig failed on overwrite: %v", err)
	}

	configFile := filepath.Join(tmpDir, ".custom-cicd", "config.yaml")
	if _, err := os.Stat(configFile); err != nil {
		t.Fatalf("expected config file to exist at %s", configFile)
	}
}
