package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/vibe-coding-labs/AtomCodeProxy/pkg/anthropic"
	"github.com/vibe-coding-labs/AtomCodeProxy/pkg/atmc"
	"github.com/vibe-coding-labs/AtomCodeProxy/pkg/dashboard"
	"github.com/vibe-coding-labs/AtomCodeProxy/pkg/openai"
	"github.com/vibe-coding-labs/AtomCodeProxy/pkg/store"
)

var (
	serveHost string
	servePort int
)

var serveCmd = &cobra.Command{
	Use:     "serve",
	Short:   "启动代理服务器",
	Long:    "启动 OpenAI/Anthropic 兼容的 API 代理服务器，将请求转发到 AtomCode Daemon。",
	GroupID: "core",
	Example: `  # 默认启动（0.0.0.0:13457）
  atomcode-proxy serve

  # 指定端口
  atomcode-proxy serve -p 8080

  # 启用调试日志
  atomcode-proxy -v serve`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServe()
	},
}

func init() {
	serveCmd.Flags().StringVarP(&serveHost, "host", "H", "0.0.0.0", "绑定地址")
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 13457, "绑定端口")
	rootCmd.AddCommand(serveCmd)
}

func runServe() error {
	daemonURL := getEnvDefault("ATOMCODE_DAEMON_URL", "http://localhost:13456")
	client := atmc.NewClient(daemonURL)

	s, err := store.Open("")
	if err != nil {
		log.Printf("Warning: store unavailable: %v", err)
	}

	if h, err := client.Health(); err != nil || h.Status != "ok" {
		log.Printf("Warning: daemon at %s not ready: %v", daemonURL, err)
	} else {
		log.Printf("Daemon connected: v%s", h.Version)
		if auth, err := client.AuthStatus(); err == nil {
			if auth.LoggedIn && auth.User != nil {
				log.Printf("Logged in as: %s", auth.User.Name)
			} else {
				log.Printf("Not logged in. Run: atomcode-proxy setup")
			}
		}
	}

	srv := openai.NewServer(client, s)
	anth := anthropic.NewHandler(client, s)
	dash := dashboard.NewHandler(s, nil, nil)

	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)
	anth.RegisterRoutes(mux)
	dash.RegisterRoutes(mux)
	mux.HandleFunc("/", dash.ServeStatic)

	handler := requestLogMiddleware(mux, s)
	if verbose {
		handler = loggingMiddleware(handler)
	}

	addr := fmt.Sprintf("%s:%d", serveHost, servePort)
	httpSrv := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	go func() {
		fmt.Println()
		fmt.Printf("  AtomCode Proxy %s\n", Version)
		fmt.Println("  ─────────────────────────────────────────────────")
		fmt.Println()
		fmt.Println("  Endpoints:")
		fmt.Println("    POST /v1/chat/completions  — Chat (OpenAI format)")
		fmt.Println("    POST /v1/messages          — Chat (Anthropic/Claude Code format)")
		fmt.Println("    GET  /v1/models            — Model list")
		fmt.Println("    GET  /health               — Health check")
		fmt.Println("    GET  /                     — Dashboard")
		fmt.Printf("    GET  /api/stats             — Dashboard API\n")
		fmt.Println()
		fmt.Println("  Claude Code setup:")
		fmt.Printf("    export ANTHROPIC_BASE_URL=http://%s\n", addr)
		fmt.Println("    export ANTHROPIC_API_KEY=atomcode")
		fmt.Println()
		fmt.Printf("  Listening on http://%s\n", addr)
		fmt.Println()

		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	httpSrv.Shutdown(ctx)
	if s != nil {
		s.Close()
	}
	log.Println("Server stopped")
	return nil
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("-> %s %s %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
		log.Printf("<- %s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

var requestCounter uint64

func requestLogMiddleware(next http.Handler, s *store.Store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		reqID := atomic.AddUint64(&requestCounter, 1)
		r = anthropic.WithRequestID(r, reqID)

		var model string
		if r.Method == "POST" && r.Body != nil {
			bodyBytes, _ := io.ReadAll(io.LimitReader(r.Body, 100<<20))
			r.Body.Close()
			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			var body map[string]any
			if json.Unmarshal(bodyBytes, &body) == nil {
				if m, ok := body["model"].(string); ok {
					model = m
				}
			}
		}

		rw := &responseWriter{ResponseWriter: w, statusCode: 200}
		next.ServeHTTP(rw, r)

		path := r.URL.Path
		if s != nil && (len(path) >= 4 && path[:4] == "/v1/" || path == "/health") {
			apiKey := r.Header.Get("x-api-key")
			if apiKey == "" {
				if a := r.Header.Get("Authorization"); len(a) > 7 && a[:7] == "Bearer " {
					apiKey = a[7:]
				}
			}
			isStream := r.URL.Query().Get("stream") != "" || path == "/v1/messages"
			latency := time.Since(start).Milliseconds()
			var errMsg string
			if rw.statusCode >= 400 {
				errMsg = fmt.Sprintf("HTTP %d on %s %s", rw.statusCode, r.Method, path)
				if body := rw.body.String(); body != "" {
					errMsg = fmt.Sprintf("%s\n%s", errMsg, body)
				}
				slog.Error("proxy error",
					"request_id", reqID,
					"status", rw.statusCode,
					"method", r.Method,
					"path", path,
					"model", model,
					"latency_ms", latency,
					"error", errMsg,
				)
			}
			s.LogRequest(apiKey, model, path, isStream, rw.statusCode, latency, errMsg, 0, 0)
		}
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(p []byte) (int, error) {
	rw.body.Write(p)
	return rw.ResponseWriter.Write(p)
}

func getEnvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}