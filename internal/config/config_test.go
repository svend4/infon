package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	// Check audio defaults
	if cfg.AudioSampleRate != 16000 {
		t.Errorf("AudioSampleRate = %d, expected 16000", cfg.AudioSampleRate)
	}

	if !cfg.EnableVAD {
		t.Error("EnableVAD should be true by default")
	}

	if !cfg.EnableNS {
		t.Error("EnableNS should be true by default")
	}

	// Check video defaults
	if cfg.VideoFPS != 15 {
		t.Errorf("VideoFPS = %d, expected 15", cfg.VideoFPS)
	}

	if !cfg.EnablePFrames {
		t.Error("EnablePFrames should be true by default")
	}

	if cfg.VideoQuality != "auto" {
		t.Errorf("VideoQuality = %s, expected 'auto'", cfg.VideoQuality)
	}

	// Check UI defaults
	if cfg.Language != "en" {
		t.Errorf("Language = %s, expected 'en'", cfg.Language)
	}

	if cfg.Theme != "dark" {
		t.Errorf("Theme = %s, expected 'dark'", cfg.Theme)
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
	}{
		{
			name:    "valid default config",
			modify:  func(c *Config) {},
			wantErr: false,
		},
		{
			name: "invalid audio sample rate",
			modify: func(c *Config) {
				c.AudioSampleRate = 12345
			},
			wantErr: true,
		},
		{
			name: "invalid video width",
			modify: func(c *Config) {
				c.VideoWidth = 100
			},
			wantErr: true,
		},
		{
			name: "invalid video height",
			modify: func(c *Config) {
				c.VideoHeight = 50
			},
			wantErr: true,
		},
		{
			name: "invalid FPS",
			modify: func(c *Config) {
				c.VideoFPS = 100
			},
			wantErr: true,
		},
		{
			name: "invalid video quality",
			modify: func(c *Config) {
				c.VideoQuality = "ultra"
			},
			wantErr: true,
		},
		{
			name: "invalid VAD sensitivity",
			modify: func(c *Config) {
				c.VADSensitivity = 1.5
			},
			wantErr: true,
		},
		{
			name: "invalid language",
			modify: func(c *Config) {
				c.Language = "xx"
			},
			wantErr: true,
		},
		{
			name: "invalid theme",
			modify: func(c *Config) {
				c.Theme = "blue"
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.modify(cfg)

			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_SaveAndLoad(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create and save config
	cfg1 := DefaultConfig()
	cfg1.Username = "testuser"
	cfg1.DisplayName = "Test User"
	cfg1.VideoFPS = 20
	cfg1.Language = "ru"

	err := cfg1.Save()
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Check file exists
	path, _ := ConfigPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Load config
	cfg2, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify values
	if cfg2.Username != "testuser" {
		t.Errorf("Username = %s, expected 'testuser'", cfg2.Username)
	}

	if cfg2.DisplayName != "Test User" {
		t.Errorf("DisplayName = %s, expected 'Test User'", cfg2.DisplayName)
	}

	if cfg2.VideoFPS != 20 {
		t.Errorf("VideoFPS = %d, expected 20", cfg2.VideoFPS)
	}

	if cfg2.Language != "ru" {
		t.Errorf("Language = %s, expected 'ru'", cfg2.Language)
	}

	// Verify defaults are preserved
	if !cfg2.EnableVAD {
		t.Error("EnableVAD should still be true")
	}
}

func TestConfig_LoadNonExistent(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Load should return default config
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Language != "en" {
		t.Errorf("Should have returned default config, got Language = %s", cfg.Language)
	}
}

func TestConfig_Merge(t *testing.T) {
	cfg1 := DefaultConfig()
	cfg1.Username = "user1"
	cfg1.VideoFPS = 15

	cfg2 := &Config{
		Username:  "user2",
		VideoFPS:  20,
		Language:  "ru",
	}

	cfg1.Merge(cfg2)

	if cfg1.Username != "user2" {
		t.Errorf("Username = %s, expected 'user2'", cfg1.Username)
	}

	if cfg1.VideoFPS != 20 {
		t.Errorf("VideoFPS = %d, expected 20", cfg1.VideoFPS)
	}

	if cfg1.Language != "ru" {
		t.Errorf("Language = %s, expected 'ru'", cfg1.Language)
	}

	// Defaults should be preserved when merge has zero values
	if !cfg1.EnableVAD {
		t.Error("EnableVAD should still be true")
	}
}

func TestConfigPath(t *testing.T) {
	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath() failed: %v", err)
	}

	if path == "" {
		t.Error("ConfigPath() returned empty string")
	}

	if !filepath.IsAbs(path) {
		t.Errorf("ConfigPath() = %s, expected absolute path", path)
	}

	if filepath.Base(path) != "config.json" {
		t.Errorf("ConfigPath() basename = %s, expected 'config.json'", filepath.Base(path))
	}

	if filepath.Base(filepath.Dir(path)) != ".tvcp" {
		t.Errorf("ConfigPath() dir = %s, expected '.tvcp'", filepath.Base(filepath.Dir(path)))
	}
}
