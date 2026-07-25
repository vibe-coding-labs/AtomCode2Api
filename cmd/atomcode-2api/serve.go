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
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/vibe-coding-labs/AtomCode2API/pkg/anthropic"
	"github.com/vibe-coding-labs/AtomCode2API/pkg/atmc"
	"github.com/vibe-coding-labs/AtomCode2API/pkg/auth"
	"github.com/vibe-coding-labs/AtomCode2API/pkg/dashboard"
	"github.com/vibe-coding-labs/AtomCode2API/pkg/openai"
	"github.com/vibe-coding-labs/AtomCode2API/pkg/store"
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
	Example: `  # 默认启动（0.0.0.0:45678）
  atomcode-2api serve

  # 指定端口
  atomcode-2api serve -p 8080

  # 启用调试日志
  atomcode-2api -v serve`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServe()
	},
}

func init() {
	serveCmd.Flags().StringVarP(&serveHost, "host", "H", "0.0.0.0", "绑定地址")
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 45678, "绑定端口")
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
				log.Printf("Not logged in. Run: atomcode-2api setup")
			}
		}
	}

	// Auto-import daemon credentials into local store on first run
	if s != nil {
		autoImportDaemon(s)
	}

	// Apply settings from store to active components
	if s != nil {
		timeoutSec := s.GetIntSetting("request_timeout", 120)
		if timeoutSec < 60 {
			timeoutSec = 60
		}
		client.SetTimeout(time.Duration(timeoutSec) * time.Second)
		log.Printf("settings: request_timeout=%ds", timeoutSec)
	}

	srv := openai.NewServer(client, s)
	anth := anthropic.NewHandler(client, s)
	dash := dashboard.NewHandler(s, nil, nil, client)

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
		fmt.Printf("  AtomCode 2API %s\n", Version)
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
		var bodyStream bool
		if r.Method == "POST" && r.Body != nil {
			bodyBytes, _ := io.ReadAll(io.LimitReader(r.Body, 100<<20))
			r.Body.Close()
			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			var body map[string]any
			if json.Unmarshal(bodyBytes, &body) == nil {
				if m, ok := body["model"].(string); ok {
					model = m
				}
				if s, ok := body["stream"].(bool); ok {
					bodyStream = s
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
			// Resolve API token to user_id so stats match the correct account
				// Resolve API token to user_id so stats match the correct account
				userID := apiKey
				if s != nil && apiKey != "" {
					acct, err := s.GetAccountByToken(apiKey)
					if err == nil && acct != nil {
						userID = acct.UserID
					} else {
						userID = "unauthenticated"
					}
				}
				isStream := bodyStream || r.URL.Query().Get("stream") != "" || path == "/v1/messages"
			latency := time.Since(start).Milliseconds()
			// Parse input/output tokens from response body if available
			inputTokens := 0
			outputTokens := 0
			if respBody := rw.body.String(); respBody != "" {
				// For non-stream JSON responses, parse directly
				var respJSON map[string]any
				if json.Unmarshal([]byte(respBody), &respJSON) == nil {
					if usage, ok := respJSON["usage"].(map[string]any); ok {
						if v, ok := usage["prompt_tokens"].(float64); ok { inputTokens = int(v) }
						if v, ok := usage["completion_tokens"].(float64); ok { outputTokens = int(v) }
						if inputTokens == 0 {
							if v, ok := usage["input_tokens"].(float64); ok { inputTokens = int(v) }
						}
						if outputTokens == 0 {
							if v, ok := usage["output_tokens"].(float64); ok { outputTokens = int(v) }
						}
					}
				}
				// For SSE stream responses, scan for usage chunk
				if inputTokens == 0 && outputTokens == 0 {
					for _, line := range strings.Split(respBody, "\n") {
						line = strings.TrimSpace(line)
						if !strings.HasPrefix(line, "data: ") { continue }
						dataStr := line[6:]
						if dataStr == "[DONE]" { continue }
						var chunk map[string]any
						if json.Unmarshal([]byte(dataStr), &chunk) == nil {
							if usage, ok := chunk["usage"].(map[string]any); ok {
								if v, ok := usage["prompt_tokens"].(float64); ok { inputTokens = int(v) }
								if v, ok := usage["completion_tokens"].(float64); ok { outputTokens = int(v) }
								if inputTokens > 0 || outputTokens > 0 { break }
							}
						}
					}
				}
			}
			var errMsg string
			if rw.statusCode >= 400 {
				errMsg = fmt.Sprintf("HTTP %d on %s %s", rw.statusCode, r.Method, path)
				if body := rw.body.String(); body != "" {
					// Truncate to 200 chars to avoid leaking secrets in logs
					sanitized := body
					if len(sanitized) > 200 {
						sanitized = sanitized[:200] + "..."
					}
					errMsg = fmt.Sprintf("%s\n%s", errMsg, sanitized)
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
			// Check if request logging is enabled in settings
			if s.GetSetting("enable_request_logging") != "false" {
				s.LogRequest(userID, model, path, isStream, rw.statusCode, latency, errMsg, inputTokens, outputTokens)
			}
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

// Flush implements http.Flusher for streaming support through the logging middleware.
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func getEnvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// autoImportDaemon reads the AtomCode daemon's local auth credentials and
// imports them into the proxy's SQLite store if no accounts exist yet.
// Also fixes the "local" account if it exists with the wrong pt_key.
func autoImportDaemon(s *store.Store) {
	accounts, err := s.ListAccounts()
	if err != nil {
		log.Printf("auto-import: list accounts failed: %v", err)
		return
	}

	creds, err := auth.LoadFromSystem()
	if err != nil {
		log.Printf("auto-import: cannot load daemon credentials: %v", err)
		return
	}
	if creds.Token == "" {
		log.Printf("auto-import: daemon not logged in (empty token)")
		return
	}

	// Check if we already have a real account with this user ID
	for _, a := range accounts {
		if a.UserID == creds.UserID && a.CredentialValid != -1 {
			return // already imported and validated
		}
	}

	// Check if user has a CodingPlan by testing the daemon chat endpoint
	nickname := creds.UserID
	if creds.UserID == "local" {
		nickname = "local"
	}

	// Try to delete stale "local" account if it exists and we have a real user ID
	if creds.UserID != "local" {
		for _, a := range accounts {
			if a.UserID == "local" {
				log.Printf("auto-import: removing stale local account (replacing with %s)", creds.UserID)
				s.RemoveAccount("local")
				break
			}
		}
	}

	if err := s.AddAccount(creds.UserID, creds.Token, nickname, true, "deepseek-v4-flash"); err != nil {
		log.Printf("auto-import: save account failed: %v", err)
		return
	}
	// Mark credentials as valid immediately since daemon is logged in with a CodingPlan
	s.SetCredentialValid(creds.UserID, true)
	log.Printf("auto-import: daemon account %s imported to SQLite store", nickname)
}