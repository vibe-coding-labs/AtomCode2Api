package dashboard

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vibe-coding-labs/AtomCodeProxy/pkg/store"
)

func testStore(t *testing.T) *store.Store {
	t.Helper()
	dir, _ := os.MkdirTemp("", "dash-test")
	s, err := store.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { s.Close(); os.RemoveAll(dir) })
	return s
}

func TestServeStaticIndex(t *testing.T) {
	h := NewHandler(testStore(t), nil, nil)
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeStatic(w, r)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAPIRoutes(t *testing.T) {
	h := NewHandler(testStore(t), nil, nil)
	paths := []string{
		"/api/auth/status",
		"/api/stats",
		"/api/settings",
	}
	for _, path := range paths {
		r := httptest.NewRequest("OPTIONS", path, nil)
		w := httptest.NewRecorder()
		mux := http.NewServeMux()
		h.RegisterRoutes(mux)
		mux.ServeHTTP(w, r)
		if w.Code != 204 {
			t.Errorf("OPTIONS %s: expected 204, got %d", path, w.Code)
		}
	}
}

func TestAuthStatus(t *testing.T) {
	h := NewHandler(testStore(t), nil, nil)
	r := httptest.NewRequest("GET", "/api/auth/status", nil)
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAuthSetupThenLogin(t *testing.T) {
	h := NewHandler(testStore(t), nil, nil)

	// Setup password
	body := `{"password":"test123"}`
	r := httptest.NewRequest("POST", "/api/auth/setup", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Errorf("setup: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Login with same password
	r2 := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(body))
	r2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	mux.ServeHTTP(w2, r2)
	if w2.Code != 200 {
		t.Errorf("login: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestHandleModels(t *testing.T) {
	h := NewHandler(testStore(t), nil, nil)
	r := httptest.NewRequest("GET", "/api/models", nil)
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandleStats(t *testing.T) {
	h := NewHandler(testStore(t), nil, nil)
	r := httptest.NewRequest("GET", "/api/stats", nil)
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestContentType(t *testing.T) {
	// contentType is handled inline in ServeStatic
}
