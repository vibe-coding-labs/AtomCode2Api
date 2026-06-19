package auth

import (
	"fmt"
	"os"
	"path/filepath"
)

const credsDir = ".atomcode-proxy"

// Credentials represents daemon authentication credentials.
type Credentials struct {
	Token  string `json:"token"`
	UserID string `json:"user_id"`
}

// SaveToken stores a token for the daemon to use.
func SaveToken(token string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, credsDir)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "daemon_token"), []byte(token), 0600)
}

// LoadToken loads a saved token.
func LoadToken() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(home, credsDir, "daemon_token"))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ClearToken removes the saved token.
func ClearToken() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	return os.Remove(filepath.Join(home, credsDir, "daemon_token"))
}

// PasswordHash is a placeholder for future bcrypt-based password auth.
func PasswordHash(password string) (string, error) {
	return password, fmt.Errorf("not implemented")
}