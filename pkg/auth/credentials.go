package auth

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const credsDir = ".atomcode-2api"

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

// ─── Keepalive ─────────────────────────────────────────────────────────────────

// DaemonCredentialRefresher implements keepalive.CredentialRefresher.
// It periodically checks the daemon auth status to keep credentials alive.
type DaemonCredentialRefresher struct {
	daemonURL string
	lastCheck time.Time
}

func NewDaemonCredentialRefresher(daemonURL string) *DaemonCredentialRefresher {
	return &DaemonCredentialRefresher{
		daemonURL: daemonURL,
	}
}

// RefreshCredentials checks if the daemon is still logged in and the token is still valid.
func (d *DaemonCredentialRefresher) RefreshCredentials() error {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(d.daemonURL + "/health")
	if err != nil {
		return fmt.Errorf("daemon unreachable: %w", err)
	}
	resp.Body.Close()

	resp, err = client.Get(d.daemonURL + "/auth/status")
	if err != nil {
		return fmt.Errorf("auth status check failed: %w", err)
	}
	defer resp.Body.Close()

	var status struct {
		LoggedIn bool `json:"logged_in"`
		Token    *struct {
			ExpiresIn int `json:"expires_in"`
		} `json:"token,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return fmt.Errorf("parse auth status: %w", err)
	}

	if !status.LoggedIn {
		return fmt.Errorf("daemon not logged in")
	}

	if status.Token != nil && status.Token.ExpiresIn < 300 {
		log.Printf("keepalive: daemon token expires in %ds, may need refresh", status.Token.ExpiresIn)
	}

	d.lastCheck = time.Now()
	return nil
}
