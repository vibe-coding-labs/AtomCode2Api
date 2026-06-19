package auth

import (
	"os"
	"testing"
)

func TestNewJWTManager(t *testing.T) {
	m := NewJWTManager("")
	if m == nil {
		t.Fatal("expected non-nil manager")
	}
}

func TestJWTTokenRoundTrip(t *testing.T) {
	m := NewJWTManager("test-secret-12345")
	token, err := m.GenerateToken("user1", "admin")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	claims, err := m.ValidateToken(token)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if claims.UserID != "user1" {
		t.Errorf("expected user1, got %s", claims.UserID)
	}
	if claims.Role != "admin" {
		t.Errorf("expected admin, got %s", claims.Role)
	}
}

func TestJWTInvalidToken(t *testing.T) {
	m := NewJWTManager("secret")
	_, err := m.ValidateToken("invalid-token")
	if err == nil {
		t.Errorf("expected error for invalid token")
	}
}

func TestJWTWrongSecret(t *testing.T) {
	m1 := NewJWTManager("secret1")
	m2 := NewJWTManager("secret2")
	token, _ := m1.GenerateToken("u", "r")
	_, err := m2.ValidateToken(token)
	if err == nil {
		t.Errorf("expected error for wrong secret")
	}
}

func TestSaveAndLoadToken(t *testing.T) {
	old := os.Getenv("HOME")
	if old == "" {
		old = os.Getenv("USERPROFILE") // Windows
	}
	_ = old
	// Use OS temp dir
	os.Setenv("HOME", t.TempDir())
	defer func() {
		if old != "" {
			os.Setenv("HOME", old)
		} else {
			os.Unsetenv("HOME")
		}
	}()

	if err := SaveToken("test-token"); err != nil {
		t.Fatalf("save: %v", err)
	}
	token, err := LoadToken()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if token != "test-token" {
		t.Errorf("expected test-token, got %s", token)
	}
	ClearToken()
	if _, err := LoadToken(); err == nil {
		t.Errorf("expected error after clear")
	}
}

func TestNewEncryptor(t *testing.T) {
	_, err := NewEncryptor()
	_ = err
}
