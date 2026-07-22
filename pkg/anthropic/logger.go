package anthropic

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/vibe-coding-labs/AtomCode2API/pkg/store"
)

// LoggerMiddleware wraps a handler with Anthropic-specific request logging.
func LoggerMiddleware(s *store.Store, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &logResponseWriter{ResponseWriter: w, statusCode: 200}
		next.ServeHTTP(rw, r)

		latency := time.Since(start).Milliseconds()
		path := r.URL.Path
		if s != nil && path == "/v1/messages" {
			apiKey := r.Header.Get("x-api-key")
			if apiKey == "" {
				if a := r.Header.Get("Authorization"); len(a) > 7 && a[:7] == "Bearer " {
					apiKey = a[7:]
				}
			}
			model := r.Header.Get("X-Request-Model")
			errMsg := ""
			if rw.statusCode >= 400 {
				errMsg = rw.body
				slog.Error("anthropic proxy error",
					"status", rw.statusCode, "path", path, "model", model,
					"latency_ms", latency, "error", errMsg)
			}
			s.LogRequest(apiKey, model, path, true, rw.statusCode, latency, errMsg, 0, 0)
		}
	})
}

type logResponseWriter struct {
	http.ResponseWriter
	statusCode int
	body       string
}

func (w *logResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *logResponseWriter) Write(p []byte) (int, error) {
	if w.statusCode >= 400 && w.body == "" && len(p) < 4096 {
		w.body = string(p)
	}
	return w.ResponseWriter.Write(p)
}
