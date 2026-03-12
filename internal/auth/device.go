package auth

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

type deviceConfig struct {
	DeviceToken string `json:"device_token"`
}

// DeviceTokenPath returns the path to the device token file.
func DeviceTokenPath() string {
	dir, _ := os.UserConfigDir()
	return filepath.Join(dir, "n26cli", "device.json")
}

// LoadOrCreateDeviceToken returns the stored device token, creating one if needed.
// Returns (token, isNew, error).
func LoadOrCreateDeviceToken() (string, bool, error) {
	path := DeviceTokenPath()

	data, err := os.ReadFile(path)
	if err == nil {
		var cfg deviceConfig
		if err := json.Unmarshal(data, &cfg); err == nil && cfg.DeviceToken != "" {
			return cfg.DeviceToken, false, nil
		}
	}

	token := uuid.New().String()
	cfg := deviceConfig{DeviceToken: token}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return "", false, err
	}

	data, _ = json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(path, data, 0600); err != nil {
		return "", false, err
	}

	return token, true, nil
}
