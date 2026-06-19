package dashboard

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"

	"github.com/vibe-coding-labs/AtomCodeProxy/pkg/store"
)

//go:embed static
var staticFiles embed.FS

// Handler serves the dashboard API and static files.
type Handler struct {
	store *store.Store
	subFS fs.FS
}

// NewHandler creates a dashboard handler.
func NewHandler(s *store.Store) *Handler {
	sub, _ := fs.Sub(staticFiles, "static")
	return &Handler{store: s, subFS: sub}
}

// RegisterRoutes registers dashboard API endpoints.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/stats", h.serveStats)
	mux.HandleFunc("/api/accounts", h.serveAccounts)
	mux.HandleFunc("/api/logs", h.serveLogs)
	mux.HandleFunc("/api/settings", h.serveSettings)
	mux.HandleFunc("/", h.serveStatic)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) serveStats(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		writeJSON(w, map[string]any{"error": "store unavailable"})
		return
	}
	stats, err := h.store.GetStats()
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, stats)
}

func (h *Handler) serveAccounts(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		writeJSON(w, []any{})
		return
	}
	accounts, err := h.store.ListAccounts()
	if err != nil {
		writeJSON(w, []any{})
		return
	}
	writeJSON(w, accounts)
}

func (h *Handler) serveLogs(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		writeJSON(w, []any{})
		return
	}
	logs, err := h.store.GetRecentLogs(50)
	if err != nil {
		writeJSON(w, []any{})
		return
	}
	writeJSON(w, logs)
}

func (h *Handler) serveSettings(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		writeJSON(w, map[string]any{})
		return
	}
	settings, _ := h.store.GetSettings()
	writeJSON(w, settings)
}

func (h *Handler) serveStatic(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(200)
		return
	}

	path := r.URL.Path
	if path == "/" || path == "" {
		path = "/index.html"
	}

	// Try embedded file first
	if h.subFS != nil {
		data, err := fs.ReadFile(h.subFS, strings.TrimPrefix(path, "/"))
		if err == nil {
			ct := contentType(path)
			w.Header().Set("Content-Type", ct)
			w.Write(data)
			return
		}
	}

	http.NotFound(w, r)
}

func contentType(path string) string {
	switch {
	case strings.HasSuffix(path, ".html"):
		return "text/html"
	case strings.HasSuffix(path, ".js"):
		return "application/javascript"
	case strings.HasSuffix(path, ".css"):
		return "text/css"
	case strings.HasSuffix(path, ".png"):
		return "image/png"
	case strings.HasSuffix(path, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(path, ".ico"):
		return "image/x-icon"
	default:
		return "text/html"
	}
}
