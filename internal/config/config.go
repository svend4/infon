package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds user preferences and settings
type Config struct {
	// Audio settings
	AudioInputDevice  string `json:"audio_input_device"`
	AudioOutputDevice string `json:"audio_output_device"`
	AudioSampleRate   int    `json:"audio_sample_rate"`
	EnableVAD         bool   `json:"enable_vad"`
	EnableNS          bool   `json:"enable_noise_suppression"`
	EnableAEC         bool   `json:"enable_echo_cancellation"`
	VADSensitivity    float64 `json:"vad_sensitivity"`
	NSSensitivity     float64 `json:"ns_sensitivity"`

	// Video settings
	VideoDevice    string `json:"video_device"`
	VideoWidth     int    `json:"video_width"`
	VideoHeight    int    `json:"video_height"`
	VideoFPS       int    `json:"video_fps"`
	EnablePFrames  bool   `json:"enable_pframes"`
	VideoQuality   string `json:"video_quality"` // "low", "medium", "high", "auto"

	// Network settings
	EnableAdaptiveBitrate bool   `json:"enable_adaptive_bitrate"`
	PreferredPort         int    `json:"preferred_port"`
	STUNServer            string `json:"stun_server"`
	TURNServer            string `json:"turn_server"`

	// UI settings
	Language       string `json:"language"` // "en", "ru", etc.
	Theme          string `json:"theme"`    // "dark", "light"
	ShowBandwidth  bool   `json:"show_bandwidth_stats"`
	ShowQuality    bool   `json:"show_quality_metrics"`

	// User profile
	Username       string `json:"username"`
	DisplayName    string `json:"display_name"`

	// Call settings
	AutoAnswer     bool   `json:"auto_answer"`
	RecordCalls    bool   `json:"record_calls"`
	RecordingsPath string `json:"recordings_path"`
}

// DefaultConfig returns a config with default values
func DefaultConfig() *Config {
	return &Config{
		// Audio defaults
		AudioInputDevice:  "default",
		AudioOutputDevice: "default",
		AudioSampleRate:   16000,
		EnableVAD:         true,
		EnableNS:          true,
		EnableAEC:         true,
		VADSensitivity:    0.7,
		NSSensitivity:     0.5,

		// Video defaults
		VideoDevice:    "default",
		VideoWidth:     640,
		VideoHeight:    480,
		VideoFPS:       15,
		EnablePFrames:  true,
		VideoQuality:   "auto",

		// Network defaults
		EnableAdaptiveBitrate: true,
		PreferredPort:         0, // Let OS choose
		STUNServer:            "stun:stun.l.google.com:19302",
		TURNServer:            "",

		// UI defaults
		Language:      "en",
		Theme:         "dark",
		ShowBandwidth: true,
		ShowQuality:   true,

		// User defaults
		Username:    "",
		DisplayName: "",

		// Call defaults
		AutoAnswer:     false,
		RecordCalls:    false,
		RecordingsPath: "",
	}
}

// ConfigPath returns the path to the config file
func ConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".tvcp")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return filepath.Join(configDir, "config.json"), nil
}

// Load loads the config from disk, or returns default if not found
func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	// If config doesn't exist, return default
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return cfg, nil
}

// Save saves the config to disk
func (c *Config) Save() error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// Validate checks if the config values are valid
func (c *Config) Validate() error {
	// Validate audio sample rate
	validRates := []int{8000, 16000, 32000, 44100, 48000}
	valid := false
	for _, rate := range validRates {
		if c.AudioSampleRate == rate {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid audio sample rate: %d (must be one of %v)", c.AudioSampleRate, validRates)
	}

	// Validate video resolution
	if c.VideoWidth < 160 || c.VideoWidth > 1920 {
		return fmt.Errorf("invalid video width: %d (must be 160-1920)", c.VideoWidth)
	}
	if c.VideoHeight < 120 || c.VideoHeight > 1080 {
		return fmt.Errorf("invalid video height: %d (must be 120-1080)", c.VideoHeight)
	}

	// Validate FPS
	if c.VideoFPS < 5 || c.VideoFPS > 60 {
		return fmt.Errorf("invalid video FPS: %d (must be 5-60)", c.VideoFPS)
	}

	// Validate video quality
	validQualities := []string{"low", "medium", "high", "auto"}
	valid = false
	for _, q := range validQualities {
		if c.VideoQuality == q {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid video quality: %s (must be one of %v)", c.VideoQuality, validQualities)
	}

	// Validate sensitivities (0.0-1.0)
	if c.VADSensitivity < 0.0 || c.VADSensitivity > 1.0 {
		return fmt.Errorf("invalid VAD sensitivity: %f (must be 0.0-1.0)", c.VADSensitivity)
	}
	if c.NSSensitivity < 0.0 || c.NSSensitivity > 1.0 {
		return fmt.Errorf("invalid NS sensitivity: %f (must be 0.0-1.0)", c.NSSensitivity)
	}

	// Validate language
	validLangs := []string{"en", "ru", "es", "fr", "de", "zh", "ja"}
	valid = false
	for _, lang := range validLangs {
		if c.Language == lang {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid language: %s (must be one of %v)", c.Language, validLangs)
	}

	// Validate theme
	if c.Theme != "dark" && c.Theme != "light" {
		return fmt.Errorf("invalid theme: %s (must be 'dark' or 'light')", c.Theme)
	}

	return nil
}

// Merge updates this config with non-zero values from another config
func (c *Config) Merge(other *Config) {
	if other.AudioInputDevice != "" {
		c.AudioInputDevice = other.AudioInputDevice
	}
	if other.AudioOutputDevice != "" {
		c.AudioOutputDevice = other.AudioOutputDevice
	}
	if other.AudioSampleRate != 0 {
		c.AudioSampleRate = other.AudioSampleRate
	}
	if other.VideoDevice != "" {
		c.VideoDevice = other.VideoDevice
	}
	if other.VideoWidth != 0 {
		c.VideoWidth = other.VideoWidth
	}
	if other.VideoHeight != 0 {
		c.VideoHeight = other.VideoHeight
	}
	if other.VideoFPS != 0 {
		c.VideoFPS = other.VideoFPS
	}
	if other.VideoQuality != "" {
		c.VideoQuality = other.VideoQuality
	}
	if other.STUNServer != "" {
		c.STUNServer = other.STUNServer
	}
	if other.TURNServer != "" {
		c.TURNServer = other.TURNServer
	}
	if other.Language != "" {
		c.Language = other.Language
	}
	if other.Theme != "" {
		c.Theme = other.Theme
	}
	if other.Username != "" {
		c.Username = other.Username
	}
	if other.DisplayName != "" {
		c.DisplayName = other.DisplayName
	}
	if other.RecordingsPath != "" {
		c.RecordingsPath = other.RecordingsPath
	}
}
