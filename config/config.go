package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Server struct {
		Port string `yaml:"port"`
		Host string `yaml:"host"`
	} `yaml:"server"`

	Librespot struct {
		DeviceName string `yaml:"device_name"`
		DeviceType string `yaml:"device_type"`
		Credentials struct {
			Type     string `yaml:"type"` // "interactive", "stored", "username_password"
			Username string `yaml:"username"`
			Password string `yaml:"password"`
		} `yaml:"credentials"`
		Audio struct {
			Bitrate              int  `yaml:"bitrate"`
			Normalisation        bool `yaml:"normalisation"`
			NormalisationPregain int  `yaml:"normalisation_pregain"`
		} `yaml:"audio"`
		StatePath string `yaml:"state_path"`
	} `yaml:"librespot"`

	Cache struct {
		Enabled   bool `yaml:"enabled"`
		MaxTracks int  `yaml:"max_tracks"`
		TTL       int  `yaml:"ttl"` // seconds
	} `yaml:"cache"`

	LogLevel string `yaml:"log_level"`
}

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	// Set defaults
	cfg := &Config{}
	cfg.Server.Port = "8080"
	cfg.Server.Host = "0.0.0.0"
	cfg.Librespot.DeviceName = "Spotify-Streamer"
	cfg.Librespot.DeviceType = "computer"
	cfg.Librespot.Credentials.Type = "interactive"
	cfg.Librespot.Audio.Bitrate = 320
	cfg.Librespot.Audio.Normalisation = true
	cfg.Librespot.Audio.NormalisationPregain = 0
	cfg.Librespot.StatePath = "./state.json"
	cfg.Cache.Enabled = true
	cfg.Cache.MaxTracks = 100
	cfg.Cache.TTL = 3600
	cfg.LogLevel = "info"

	// Check if config file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Config file doesn't exist, use defaults
		return cfg, nil
	}

	// Read config file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}
