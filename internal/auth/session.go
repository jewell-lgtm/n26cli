package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Session holds persisted auth state.
type Session struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// SessionPath returns the path to the session file.
func SessionPath() string {
	dir, _ := os.UserConfigDir()
	return filepath.Join(dir, "n26cli", "session.json")
}

// LoadSession reads and validates the stored session.
// Returns nil if no valid session exists.
func LoadSession() *Session {
	data, err := os.ReadFile(SessionPath())
	if err != nil {
		return nil
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil
	}
	if s.AccessToken == "" {
		return nil
	}
	return &s
}

// IsValid returns true if the session hasn't expired (with 60s buffer).
func (s *Session) IsValid() bool {
	if s == nil {
		return false
	}
	return time.Now().Add(60 * time.Second).Before(s.ExpiresAt)
}

// MinutesRemaining returns minutes until expiry.
func (s *Session) MinutesRemaining() int {
	if s == nil {
		return 0
	}
	d := time.Until(s.ExpiresAt)
	if d < 0 {
		return 0
	}
	return int(d.Minutes())
}

// Save persists the session to disk with 0600 permissions.
func (s *Session) Save() error {
	path := SessionPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling session: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}

// Delete removes the session file.
func Delete() error {
	return os.Remove(SessionPath())
}
