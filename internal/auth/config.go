package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds persistent CLI settings.
type Config struct {
	OutputDir string `json:"output_dir,omitempty"`
}

// ConfigPath returns the path to the config file.
func ConfigPath() string {
	dir, _ := os.UserConfigDir()
	return filepath.Join(dir, "n26", "config.json")
}

// LoadConfig reads the config file. Returns defaults if missing.
func LoadConfig() *Config {
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		return &Config{}
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return &Config{}
	}
	return &c
}

// Save persists the config to disk.
func (c *Config) Save() error {
	path := ConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
