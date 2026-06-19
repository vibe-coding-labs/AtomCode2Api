package dashboard

import (
	"net/http/httptest"
	"testing"
)

func TestServeIndex(t *testing.T) {
	h := NewHandler(nil)
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.serveStatic(w, r)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if len(w.Body.Bytes()) == 0 {
		t.Errorf("expected non-empty body")
	}
}

func TestServeOptions(t *testing.T) {
	h := NewHandler(nil)
	r := httptest.NewRequest("OPTIONS", "/api/stats", nil)
	w := httptest.NewRecorder()
	h.serveStats(w, r)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAPIStatsNoStore(t *testing.T) {
	h := NewHandler(nil)
	r := httptest.NewRequest("GET", "/api/stats", nil)
	w := httptest.NewRecorder()
	h.serveStats(w, r)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAPILogsNoStore(t *testing.T) {
	h := NewHandler(nil)
	r := httptest.NewRequest("GET", "/api/logs", nil)
	w := httptest.NewRecorder()
	h.serveLogs(w, r)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAPISettingsNoStore(t *testing.T) {
	h := NewHandler(nil)
	r := httptest.NewRequest("GET", "/api/settings", nil)
	w := httptest.NewRecorder()
	h.serveSettings(w, r)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestContentType(t *testing.T) {
	tests := []struct{ path, expected string }{
		{"/", "text/html"},
		{"/index.html", "text/html"},
		{"/app.js", "application/javascript"},
		{"/style.css", "text/css"},
	}
	for _, tc := range tests {
		if ct := contentType(tc.path); ct != tc.expected {
			t.Errorf("contentType(%q) = %q, want %q", tc.path, ct, tc.expected)
		}
	}
}