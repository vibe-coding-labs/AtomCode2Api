package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	storeDir = ".atomcode-proxy"
	keyFile  = ".auth_key"
)

var (
	defaultStoreDir string
)

func init() {
	home, _ := os.UserHomeDir()
	defaultStoreDir = filepath.Join(home, storeDir)
}

// ─── AES-GCM Encryption ─────────────────────────────────────────────────────

type Encryptor struct {
	aead cipher.AEAD
}

func NewEncryptor() (*Encryptor, error) {
	if err := os.MkdirAll(defaultStoreDir, 0700); err != nil {
		return nil, err
	}
	keyPath := filepath.Join(defaultStoreDir, keyFile)
	data, err := os.ReadFile(keyPath)
	if err != nil {
		key := make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, key); err != nil {
			return nil, err
		}
		if err := os.WriteFile(keyPath, []byte(hex.EncodeToString(key)), 0600); err != nil {
			return nil, err
		}
		data = []byte(hex.EncodeToString(key))
	}
	key, err := hex.DecodeString(string(data))
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("invalid encryption key")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Encryptor{aead: aead}, nil
}

func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	nonce := make([]byte, e.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := e.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
}

func (e *Encryptor) Decrypt(ciphertext string) (string, error) {
	data, err := hex.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	nonceSize := e.aead.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ct := data[:nonceSize], data[nonceSize:]
	plaintext, err := e.aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// ─── JWT ─────────────────────────────────────────────────────────────────────

type JWTManager struct {
	secret   []byte
	issuer   string
	duration time.Duration
}

func NewJWTManager(secret string) *JWTManager {
	if secret == "" {
		b := make([]byte, 32)
		io.ReadFull(rand.Reader, b)
		secret = hex.EncodeToString(b)
	}
	return &JWTManager{
		secret:   []byte(secret),
		issuer:   "atomcode-proxy",
		duration: 24 * time.Hour,
	}
}

type Claims struct {
	jwt.RegisteredClaims
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

func (m *JWTManager) GenerateToken(userID, role string) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.duration)),
		},
		UserID: userID,
		Role:   role,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

func (m *JWTManager) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}