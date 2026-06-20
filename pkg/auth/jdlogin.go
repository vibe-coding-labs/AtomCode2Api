package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// QRSession holds the state for a QR login flow.
type QRSession struct {
	SessionID string
	PtKey     string
	UserID    string
	RealName  string
	ExpiresAt time.Time
}

type QRVerifyNeededError struct {
	VerifyURL string
	RiskCode  string
}

func (e *QRVerifyNeededError) Error() string {
	return fmt.Sprintf("verification required: %s (risk=%s)", e.VerifyURL, e.RiskCode)
}

var qrSessions = make(map[string]*QRSession)

const (
	jdhBaseURL = "https://jdh-sdk.jd.com"
	appID      = "jdh_joycode"
)

// QRInit creates a QR login session and returns the session ID and QR image base64.
func QRInit() (string, string, error) {
	// Generate RSA key pair for the QR challenge
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", fmt.Errorf("generate key: %w", err)
	}
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return "", "", err
	}
	pubKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubKeyBytes})

	// Request QR code from JD
	body := fmt.Sprintf(`{"appId":"%s","pubKey":"%s"}`, appID, strings.ReplaceAll(string(pubKeyPEM), "\n", "\\n"))
	resp, err := http.Post(jdhBaseURL+"/api/qr/QRCodeApply", "application/json", strings.NewReader(body))
	if err != nil {
		return "", "", fmt.Errorf("qr apply: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			SessionID string `json:"sessionId"`
			QRImage   string `json:"qrImage"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", "", fmt.Errorf("parse qr response: %w", err)
	}
	if result.Code != 0 {
		return "", "", fmt.Errorf("qr api error (code=%d): %s", result.Code, result.Msg)
	}

	qrSessions[result.Data.SessionID] = &QRSession{
		SessionID: result.Data.SessionID,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	return result.Data.SessionID, result.Data.QRImage, nil
}

// QRPollStatus polls the QR login status.
func QRPollStatus(sessionID string) (string, *QRSession, error) {
	sess, ok := qrSessions[sessionID]
	if !ok {
		return "", nil, fmt.Errorf("session not found")
	}
	if time.Now().After(sess.ExpiresAt) {
		return "expired", nil, nil
	}

	resp, err := http.Post(jdhBaseURL+"/api/qr/QRCodeCheckStatus",
		"application/json",
		strings.NewReader(fmt.Sprintf(`{"sessionId":"%s","appId":"%s"}`, sessionID, appID)),
	)
	if err != nil {
		return "", nil, fmt.Errorf("qr check: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Status     string `json:"status"`
			VerifyURL  string `json:"verifyUrl"`
			RiskCode   string `json:"riskCode"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", nil, fmt.Errorf("parse: %w", err)
	}

	switch result.Data.Status {
	case "SCANNED":
		return "scanned", nil, nil
	case "CONFIRMED":
		return "confirmed", sess, nil
	case "VERIFICATION_REQUIRED":
		return "", nil, &QRVerifyNeededError{
			VerifyURL: result.Data.VerifyURL,
			RiskCode:  result.Data.RiskCode,
		}
	default:
		return result.Data.Status, nil, nil
	}
}

// LoadFromSystem loads JoyCode credentials from the local system.
// This is AtomCode-specific — reads from ~/.atomcode/ config.
func LoadFromSystem() (*Credentials, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	paths := []string{
		home + "/.atomcode/auth.toml",
		home + "/.atomcode/config.toml",
	}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		// Simple TOML-like parse for api_key
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "api_key") || strings.HasPrefix(line, "token") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					val := strings.TrimSpace(parts[1])
					val = strings.Trim(val, `"`)
					if val != "" {
						return &Credentials{Token: val, UserID: "local"}, nil
					}
				}
			}
		}
	}
	return nil, fmt.Errorf("no credentials found in %s", strings.Join(paths, ", "))
}

// GenerateToken creates a JWT-like signed token using the secret.
func GenerateToken(userID, secret string, expiry time.Duration) (string, error) {
	payload := fmt.Sprintf(`{"sub":"%s","exp":%d}`, userID, time.Now().Add(expiry).Unix())
	mac := hmacSHA256(payload, secret)
	token := base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + base64.RawURLEncoding.EncodeToString(mac)
	return token, nil
}

func hmacSHA256(data, key string) []byte {
	m := hmac.New(sha256.New, []byte(key))
	m.Write([]byte(data))
	return m.Sum(nil)
}

func init() {
	_ = url.QueryEscape
}