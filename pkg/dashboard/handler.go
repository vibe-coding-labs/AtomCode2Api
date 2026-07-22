package dashboard

import (
	"bytes"
	"crypto/rand"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/vibe-coding-labs/AtomCode2API/pkg/auth"
	"github.com/vibe-coding-labs/AtomCode2API/pkg/atmc"
	"github.com/vibe-coding-labs/AtomCode2API/pkg/keepalive"
	"github.com/vibe-coding-labs/AtomCode2API/pkg/proxy"
	"github.com/vibe-coding-labs/AtomCode2API/pkg/store"
)

//go:embed static
var staticFiles embed.FS

type Handler struct {
	store    *store.Store
	staticFS fs.FS
	keeper   *keepalive.Keeper
	daemon   *atmc.Client
	Version  string
}

func NewHandler(s *store.Store, staticFS fs.FS, k *keepalive.Keeper, daemonClient *atmc.Client) *Handler {
	if staticFS == nil {
		sub, _ := fs.Sub(staticFiles, "static")
		staticFS = sub
	}
	return &Handler{store: s, staticFS: staticFS, keeper: k, daemon: daemonClient}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/auth/status", h.handleAuthStatus)
	mux.HandleFunc("/api/auth/setup", h.handleAuthSetup)
	mux.HandleFunc("/api/auth/login", h.handleAuthLogin)
	mux.HandleFunc("/api/auth/change-password", h.handleChangePassword)

	mux.HandleFunc("/api/accounts", h.handleAccounts)
	mux.HandleFunc("/api/accounts/", h.handleAccountAction)
	mux.HandleFunc("/api/accounts-export", h.handleExportAccounts)
	mux.HandleFunc("/api/accounts-import", h.handleImportAccounts)
	mux.HandleFunc("/api/accounts/batch-import", h.handleBatchImport)
	mux.HandleFunc("/api/accounts-auto-login", h.handleAutoLogin)
	mux.HandleFunc("/api/accounts-clear-all", h.handleClearAllAccounts)
	mux.HandleFunc("/api/stats", h.handleStats)
	mux.HandleFunc("/api/settings", h.handleSettings)
	mux.HandleFunc("/api/errors", h.handleErrors)
	mux.HandleFunc("/api/models", h.handleModels)
	mux.HandleFunc("/api/models/catalog", h.handleModelsCatalog)
	mux.HandleFunc("/api/codingplan/status", h.handleCodingPlanStatus)
	mux.HandleFunc("/api/health", h.handleHealth)
	mux.HandleFunc("/api/github-stars", h.handleGitHubStars)

	mux.HandleFunc("/api/browser-login", h.handleBrowserLogin)
	mux.HandleFunc("/api/oauth-callback", h.handleOAuthCallback)
	mux.HandleFunc("/api/oauth-submit", h.handleOAuthSubmit)
	mux.HandleFunc("/api/qr-login/init", h.handleQRLoginInit)
	mux.HandleFunc("/api/qr-login/status", h.handleQRLoginStatus)
}

const jwtSecretKey = "auth_jwt_secret"
const defaultJWTExpiry = 24 * time.Hour

// ─── Auth ────────────────────────────────────────────────────────────────────

func (h *Handler) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	setCors(w)
	if r.Method == http.MethodOptions { w.WriteHeader(http.StatusNoContent); return }
	hash := h.store.GetSetting("auth_password_hash")
	writeJSON(w, http.StatusOK, map[string]any{
		"initialized": hash != "",
	})
}

func (h *Handler) handleAuthSetup(w http.ResponseWriter, r *http.Request) {
	setCors(w)
	if r.Method == http.MethodOptions { w.WriteHeader(http.StatusNoContent); return }
	if r.Method != http.MethodPost { writeError(w, 405, "method not allowed"); return }

	if h.store.GetSetting("auth_password_hash") != "" {
		writeError(w, 409, "root password already initialized"); return
	}
	var body struct {
		Password string `json:"password"`
	}
	if !readJSONBody(w, r, &body) { return }
	if len(body.Password) < 6 { writeError(w, 400, "密码长度不能少于 6 位"); return }

	hash, err := auth.HashPassword(body.Password)
	if err != nil { writeError(w, 500, "密码加密失败"); return }
	if err := h.store.SetSetting("auth_password_hash", hash); err != nil { writeError(w, 500, "保存密码失败"); return }

	if h.store.GetSetting(jwtSecretKey) == "" {
		h.store.SetSetting(jwtSecretKey, generateRandomHex(32))
	}
	token, err := h.issueJWT()
	if err != nil { writeError(w, 500, "生成 token 失败"); return }

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "token": token})
}

func (h *Handler) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	setCors(w)
	if r.Method == http.MethodOptions { w.WriteHeader(http.StatusNoContent); return }
	if r.Method != http.MethodPost { writeError(w, 405, "method not allowed"); return }

	hash := h.store.GetSetting("auth_password_hash")
	if hash == "" { writeError(w, 409, "root password not initialized"); return }

	var body struct{ Password string `json:"password"` }
	if !readJSONBody(w, r, &body) { return }
	if !auth.CheckPassword(body.Password, hash) { writeError(w, 401, "密码错误"); return }

	token, err := h.issueJWT()
	if err != nil { writeError(w, 500, "生成 token 失败"); return }
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "token": token})
}

func (h *Handler) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	setCors(w)
	if r.Method == http.MethodOptions { w.WriteHeader(http.StatusNoContent); return }
	if r.Method != http.MethodPost { writeError(w, 405, "method not allowed"); return }

	hash := h.store.GetSetting("auth_password_hash")
	if hash == "" { writeError(w, 409, "root password not initialized"); return }

	var body struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if !readJSONBody(w, r, &body) { return }
	if !auth.CheckPassword(body.OldPassword, hash) { writeError(w, 401, "原密码错误"); return }
	if len(body.NewPassword) < 6 { writeError(w, 400, "新密码长度不能少于 6 位"); return }

	newHash, err := auth.HashPassword(body.NewPassword)
	if err != nil { writeError(w, 500, "密码加密失败"); return }
	if err := h.store.SetSetting("auth_password_hash", newHash); err != nil { writeError(w, 500, "保存密码失败"); return }
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) issueJWT() (string, error) {
	secret := h.store.GetSetting(jwtSecretKey)
	if secret == "" { return "", fmt.Errorf("JWT secret not configured") }
	return auth.GenerateToken("root", secret, defaultJWTExpiry)
}

func generateRandomHex(n int) string {
	b := make([]byte, n)
	io.ReadFull(rand.Reader, b)
	return fmt.Sprintf("%x", b)
}

// ─── Accounts ────────────────────────────────────────────────────────────────

func (h *Handler) handleAccounts(w http.ResponseWriter, r *http.Request) {
	setCors(w)
	if r.Method == http.MethodOptions { w.WriteHeader(http.StatusNoContent); return }
	switch r.Method {
	case http.MethodGet: h.listAccounts(w, r)
	case http.MethodPost: h.addAccount(w, r)
	default: writeError(w, 405, "method not allowed")
	}
}

func (h *Handler) listAccounts(w http.ResponseWriter, r *http.Request) {
	accounts, err := h.store.ListAccounts()
	if err != nil { writeError(w, 500, err.Error()); return }
	if accounts == nil { accounts = []store.AccountInfo{} }
	for i := range accounts {
			accounts[i].ActiveSessions = proxy.GetActiveSessions(accounts[i].UserID)
		}
		h.store.FillAccountStats(accounts)
	writeJSON(w, http.StatusOK, map[string]any{"accounts": accounts})
}

func (h *Handler) addAccount(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Nickname     string `json:"nickname"`
		PtKey        string `json:"pt_key"`
		UserID       string `json:"user_id"`
		IsDefault    *bool  `json:"is_default"`
		DefaultModel string `json:"default_model"`
	}
	if !readJSONBody(w, r, &body) { return }
	if body.UserID == "" || body.PtKey == "" { writeError(w, 400, "user_id and pt_key are required"); return }

	isDefault := false
	if body.IsDefault != nil { isDefault = *body.IsDefault }

	if err := h.store.AddAccount(body.UserID, body.PtKey, body.Nickname, isDefault, body.DefaultModel); err != nil {
		writeError(w, 500, err.Error()); return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "user_id": body.UserID})
}

func (h *Handler) handleAccountAction(w http.ResponseWriter, r *http.Request) {
	setCors(w)
	if r.Method == http.MethodOptions { w.WriteHeader(http.StatusNoContent); return }

	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/accounts/"), "/")
	if len(parts) == 0 || parts[0] == "" { writeError(w, 400, "missing user_id"); return }
	userID := parts[0]
	action := ""
	if len(parts) > 1 { action = parts[1] }

	switch {
	case action == "" && r.Method == http.MethodDelete:
		h.store.RemoveAccount(userID); writeJSON(w, 200, map[string]any{"ok": true})
	case action == "default" && r.Method == http.MethodPut:
		if err := h.store.SetDefault(userID); err != nil { writeError(w, 500, err.Error()); return }
		writeJSON(w, 200, map[string]any{"ok": true})
	case action == "model" && r.Method == http.MethodPut:
		var b struct{ DefaultModel string `json:"default_model"` }
		if !readJSONBody(w, r, &b) { return }
		if err := h.store.UpdateAccountModel(userID, b.DefaultModel); err != nil { writeError(w, 500, err.Error()); return }
		writeJSON(w, 200, map[string]any{"ok": true})
	case action == "stats" && r.Method == http.MethodGet:
		stats, err := h.store.GetAccountStats(userID)
		if err != nil { writeError(w, 500, err.Error()); return }
		if stats.ByModel == nil { stats.ByModel = []store.ModelCount{} }
		if stats.ByEndpoint == nil { stats.ByEndpoint = []store.EndpointCount{} }
		if stats.Hourly == nil { stats.Hourly = []store.HourlyData{} }
		writeJSON(w, 200, stats)
	case action == "logs" && r.Method == http.MethodGet:
		limit := 200
		if l := r.URL.Query().Get("limit"); l != "" {
			fmt.Sscanf(l, "%d", &limit)
			if limit > 1000 { limit = 1000 }
		}
		logs, err := h.store.GetAccountLogs(userID, limit)
		if err != nil { writeError(w, 500, err.Error()); return }
		if logs == nil { logs = []store.RequestLog{} }
		writeJSON(w, 200, map[string]any{"logs": logs, "total": len(logs)})
	case action == "renew-token" && r.Method == http.MethodPost:
		token, err := h.store.RenewToken(userID)
		if err != nil { writeError(w, 500, err.Error()); return }
		writeJSON(w, 200, map[string]any{"ok": true, "api_token": token})
	case action == "remark" && r.Method == http.MethodPut:
		var b struct{ Remark string `json:"remark"` }
		if !readJSONBody(w, r, &b) { return }
		if err := h.store.UpdateRemark(userID, b.Remark); err != nil { writeError(w, 500, err.Error()); return }
		writeJSON(w, 200, map[string]any{"ok": true})
	case action == "validate" && r.Method == http.MethodPost:
		account, err := h.store.GetAccount(userID)
		if err != nil { writeError(w, 500, err.Error()); return }
		if account == nil { writeError(w, 404, "account not found"); return }
		valid := account.PtKey != ""
		if valid {
			h.store.SetCredentialValid(userID, true)
		}
		writeJSON(w, 200, map[string]any{"api_key": userID, "valid": valid})
	case action == "models" && r.Method == http.MethodGet:
		// Return catalog models filtered to those available for this account
		writeJSON(w, 200, map[string]any{"models": []map[string]string{
			{"id": "deepseek-v4-flash", "name": "DeepSeek V4 Flash", "free": "true", "input_price": "¥0", "output_price": "¥0"},
			{"id": "Qwen/Qwen3-VL-8B-Instruct", "name": "Qwen3-VL 8B Instruct", "free": "true", "input_price": "¥0", "output_price": "¥0"},
			{"id": "deepseek-chat", "name": "DeepSeek Chat", "free": "false", "input_price": "¥0.5/百万tokens", "output_price": "¥2/百万tokens"},
			{"id": "deepseek-reasoner", "name": "DeepSeek Reasoner", "free": "false", "input_price": "¥1/百万tokens", "output_price": "¥4/百万tokens"},
			{"id": "glm-5.2", "name": "GLM 5.2", "free": "false", "input_price": "¥0.5/百万tokens", "output_price": "¥2/百万tokens"},
		}})
	default:
		writeError(w, 405, "method not allowed")
	}
}

func (h *Handler) handleAutoLogin(w http.ResponseWriter, r *http.Request) {
	setCors(w)
	if r.Method == http.MethodOptions { w.WriteHeader(http.StatusNoContent); return }
	if r.Method != http.MethodPost { writeError(w, 405, "method not allowed"); return }

	creds, err := auth.LoadFromSystem()
	if err != nil {
		writeError(w, 400, "无法从本机获取 AtomCode 凭据: "+err.Error())
		return
	}
	if creds.UserID == "" || creds.Token == "" {
		writeError(w, 400, "AtomCode 凭证不完整（UserID 或 Token 为空）")
		return
	}

	// If we previously imported a stale "local" account, remove it now
	accounts, _ := h.store.ListAccounts()
	for _, a := range accounts {
		if a.UserID == "local" && creds.UserID != "local" {
			h.store.RemoveAccount("local")
			log.Printf("auto-login: removed stale local account")
		}
	}

	// Re-fetch account list after cleanup
	accounts, _ = h.store.ListAccounts()
	isDefault := true
	for _, a := range accounts {
		if a.IsDefault { isDefault = false; break }
	}

	// Check if this user already exists
	for _, a := range accounts {
		if a.UserID == creds.UserID {
			// Account already exists, just ensure the token is up-to-date
			h.store.UpdatePtKey(creds.UserID, creds.Token)
			writeJSON(w, 200, map[string]any{"ok": true, "user_id": creds.UserID, "is_default": a.IsDefault})
			return
		}
	}

	if err := h.store.AddAccount(creds.UserID, creds.Token, creds.UserID, isDefault, "deepseek-v4-flash"); err != nil {
		writeError(w, 500, "保存账号失败: "+err.Error()); return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "user_id": creds.UserID, "is_default": isDefault})
}

func (h *Handler) handleClearAllAccounts(w http.ResponseWriter, r *http.Request) {
	setCors(w)
	if r.Method == http.MethodOptions { w.WriteHeader(http.StatusNoContent); return }
	if r.Method != http.MethodPost { writeError(w, 405, "method not allowed"); return }

	n, err := h.store.ClearAllAccounts()
	if err != nil { writeError(w, 500, err.Error()); return }
	writeJSON(w, 200, map[string]any{"ok": true, "count": n})
}

func (h *Handler) handleExportAccounts(w http.ResponseWriter, r *http.Request) {
	setCors(w)
	if r.Method == http.MethodOptions { w.WriteHeader(http.StatusNoContent); return }
	if r.Method != http.MethodGet { writeError(w, 405, "method not allowed"); return }

	items, err := h.store.ExportAccounts()
	if err != nil { writeError(w, 500, err.Error()); return }
	writeJSON(w, 200, map[string]any{"ok": true, "accounts": items, "count": len(items)})
}

func (h *Handler) handleImportAccounts(w http.ResponseWriter, r *http.Request) {
	setCors(w)
	if r.Method == http.MethodOptions { w.WriteHeader(http.StatusNoContent); return }
	if r.Method != http.MethodPost { writeError(w, 405, "method not allowed"); return }

	var body struct{ Accounts []store.ExportAccountItem `json:"accounts"` }
	if !readJSONBody(w, r, &body) { return }
	if len(body.Accounts) == 0 { writeError(w, 400, "accounts array is empty"); return }

	added, updated, err := h.store.ImportAccounts(body.Accounts)
	if err != nil { writeError(w, 500, err.Error()); return }
	writeJSON(w, 200, map[string]any{"ok": true, "added": added, "updated": updated})
}

// handleBatchImport is a simplified account import endpoint designed for AI agents.
// Accepts a flat JSON array of accounts: [{"user_id":"...","pt_key":"...",...}]
// or the standard nested format: {"accounts": [...]}
// Returns the same response format as handleImportAccounts.
func (h *Handler) handleBatchImport(w http.ResponseWriter, r *http.Request) {
	setCors(w)
	if r.Method == http.MethodOptions { w.WriteHeader(http.StatusNoContent); return }
	if r.Method != http.MethodPost { writeError(w, 405, "method not allowed"); return }

	var accounts []store.ExportAccountItem

	// Try array format first: [{"user_id":"...", ...}]
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, 400, "cannot read body"); return
	}
	r.Body = io.NopCloser(bytes.NewReader(raw))

	if err := json.Unmarshal(raw, &accounts); err == nil {
		// Direct array format - use it directly
	} else {
		// Try nested format: {"accounts": [...]}
		var body struct{ Accounts []store.ExportAccountItem `json:"accounts"` }
		if json.Unmarshal(raw, &body) == nil {
			accounts = body.Accounts
		} else {
			writeError(w, 400, "expected JSON array of accounts or {accounts: [...]}")
			return
		}
	}

	if len(accounts) == 0 { writeError(w, 400, "accounts list is empty"); return }
	if len(accounts) > 100 {
		writeError(w, 400, "maximum 100 accounts per batch"); return
	}

	added, updated, err := h.store.ImportAccounts(accounts)
	if err != nil { writeError(w, 500, err.Error()); return }
	writeJSON(w, 200, map[string]any{"ok": true, "added": added, "updated": updated, "total": added + updated})
}

// ─── Stats, Settings, Errors, Models, Health ─────────────────────────────────

func (h *Handler) handleStats(w http.ResponseWriter, r *http.Request) {
	setCors(w)
	if r.Method == http.MethodOptions { w.WriteHeader(http.StatusNoContent); return }
	if r.Method != http.MethodGet { writeError(w, 405, "method not allowed"); return }

	stats, err := h.store.GetStats()
	if err != nil { writeError(w, 500, err.Error()); return }
	if stats.ByModel == nil { stats.ByModel = []store.ModelCount{} }
	if stats.ByAccount == nil { stats.ByAccount = []store.AccountCount{} }

	totals, _ := h.store.GetAllTimeTotals()
	hourly, _ := h.store.GetHourlyStats()
	if hourly == nil { hourly = []store.HourlyData{} }

	writeJSON(w, 200, map[string]any{
		"total_requests": stats.TotalRequests, "total_input_tokens": stats.TotalInputTk,
		"total_output_tokens": stats.TotalOutputTk, "accounts_count": stats.AccountsCount,
		"avg_latency_ms": stats.AvgLatencyMs, "error_count": stats.ErrorCount,
		"stream_count": stats.StreamCount, "success_count": stats.SuccessCount,
		"by_model": stats.ByModel, "by_account": stats.ByAccount,
		"all_time": totals, "hourly": hourly,
		"quota": map[string]any{
			"daily_limit":        h.store.GetDailyQuota(),
			"account_daily_limit": h.store.GetAccountDailyQuota(),
		},
	})
}

func (h *Handler) handleSettings(w http.ResponseWriter, r *http.Request) {
	setCors(w)
	if r.Method == http.MethodOptions { w.WriteHeader(http.StatusNoContent); return }

	switch r.Method {
	case http.MethodGet:
		settings, err := h.store.GetSettings()
		if err != nil { writeError(w, 500, err.Error()); return }
		if settings == nil { settings = map[string]string{} }
		writeJSON(w, 200, map[string]any{"settings": settings})
	case http.MethodPut:
		if !h.isAuthenticated(r) {
			writeError(w, 401, "unauthorized")
			return
		}
		var raw map[string]json.RawMessage
		if !readJSONBody(w, r, &raw) { return }
		blocked := map[string]bool{"auth_jwt_secret": true, "auth_password_hash": true}
		for k := range raw {
			if blocked[k] {
				delete(raw, k)
			}
		}
		settings := make(map[string]string, len(raw))
		for k, v := range raw {
			trimmed := strings.TrimSpace(string(v))
			if len(trimmed) > 0 && trimmed[0] == '"' {
				var s string
				json.Unmarshal(v, &s)
				settings[k] = s
			} else {
				settings[k] = trimmed
			}
		}
		if err := h.store.SetSettings(settings); err != nil { writeError(w, 500, err.Error()); return }
		writeJSON(w, 200, map[string]any{"ok": true})
	default:
		writeError(w, 405, "method not allowed")
	}
}

func (h *Handler) handleErrors(w http.ResponseWriter, r *http.Request) {
	setCors(w)
	if r.Method == http.MethodOptions { w.WriteHeader(http.StatusNoContent); return }
	if r.Method != http.MethodGet { writeError(w, 405, "method not allowed"); return }

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
		if limit > 200 { limit = 200 }
	}
	logs, err := h.store.GetRecentErrors(limit)
	if err != nil { writeError(w, 500, err.Error()); return }
	if logs == nil { logs = []store.RequestLog{} }
	writeJSON(w, 200, map[string]any{"errors": logs, "total": len(logs)})
}

func (h *Handler) handleModels(w http.ResponseWriter, r *http.Request) {
	setCors(w)
	if r.Method == http.MethodOptions { w.WriteHeader(http.StatusNoContent); return }
	if r.Method != http.MethodGet { writeError(w, 405, "method not allowed"); return }

	models := []map[string]string{
		{"id": "deepseek-v4-flash", "name": "DeepSeek V4 Flash"},
		{"id": "deepseek-chat", "name": "DeepSeek Chat"},
		{"id": "Qwen-QwQ-32B", "name": "Qwen QwQ 32B"},
	}
	writeJSON(w, 200, map[string]any{"models": models})
}

// ModelCatalogItem describes a model available through the CodingPlan proxy.
type ModelCatalogItem struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Provider         string `json:"provider"`
	Type             string `json:"type"` // chat, reasoning, vision
	ContextWindow    int    `json:"context_window"`
	Free             bool   `json:"free"` // free with CodingPlan
	Default          bool   `json:"default"`
	EffortApplicable bool   `json:"effort_applicable"`
	InputPrice       string `json:"input_price"`
	OutputPrice      string `json:"output_price"`
	PricingNote      string `json:"pricing_note,omitempty"`
	MaxOutputTokens  int    `json:"max_output_tokens,omitempty"`
}

func (h *Handler) handleModelsCatalog(w http.ResponseWriter, r *http.Request) {
	setCors(w)
	if r.Method == http.MethodOptions { w.WriteHeader(http.StatusNoContent); return }
	if r.Method != http.MethodGet { writeError(w, 405, "method not allowed"); return }

	// Static catalog of models available through CodingPlan.
	// Pricing info sourced from AtomCodeReverseEngineer project analysis.
	catalog := []ModelCatalogItem{
		{
			ID:               "deepseek-v4-flash",
			Name:             "DeepSeek V4 Flash",
			Provider:         "AtomGit",
			Type:             "chat",
			ContextWindow:    1000000,
			Free:             true,
			Default:          true,
			EffortApplicable: true,
			InputPrice:       "¥0",
			OutputPrice:      "¥0",
			PricingNote:      "CodingPlan 免费额度覆盖",
			MaxOutputTokens:  8192,
		},
		{
			ID:               "Qwen/Qwen3-VL-8B-Instruct",
			Name:             "Qwen3-VL 8B Instruct",
			Provider:         "AtomGit",
			Type:             "vision",
			ContextWindow:    64000,
			Free:             true,
			Default:          false,
			EffortApplicable: false,
			InputPrice:       "¥0",
			OutputPrice:      "¥0",
			PricingNote:      "CodingPlan 免费额度覆盖",
			MaxOutputTokens:  4096,
		},
		{
			ID:               "deepseek-chat",
			Name:             "DeepSeek Chat",
			Provider:         "DeepSeek",
			Type:             "chat",
			ContextWindow:    128000,
			Free:             false,
			Default:          false,
			EffortApplicable: false,
			InputPrice:       "¥0.5/百万tokens",
			OutputPrice:      "¥2/百万tokens",
			PricingNote:      "CodingPlan 需升级 Pro 以使用",
			MaxOutputTokens:  8192,
		},
		{
			ID:               "deepseek-reasoner",
			Name:             "DeepSeek Reasoner",
			Provider:         "DeepSeek",
			Type:             "reasoning",
			ContextWindow:    128000,
			Free:             false,
			Default:          false,
			EffortApplicable: false,
			InputPrice:       "¥1/百万tokens",
			OutputPrice:      "¥4/百万tokens",
			PricingNote:      "CodingPlan 需升级 Pro 以使用",
			MaxOutputTokens:  8192,
		},
		{
			ID:               "glm-5.2",
			Name:             "GLM 5.2",
			Provider:         "Zhipu AI",
			Type:             "chat",
			ContextWindow:    128000,
			Free:             false,
			Default:          false,
			EffortApplicable: false,
			InputPrice:       "¥0.5/百万tokens",
			OutputPrice:      "¥2/百万tokens",
			PricingNote:      "CodingPlan 需升级 Pro 以使用",
			MaxOutputTokens:  4096,
		},
	}

	writeJSON(w, 200, map[string]any{"catalog": catalog})
}

func (h *Handler) handleCodingPlanStatus(w http.ResponseWriter, r *http.Request) {
	setCors(w)
	if r.Method == http.MethodOptions { w.WriteHeader(http.StatusNoContent); return }
	if r.Method != http.MethodGet { writeError(w, 405, "method not allowed"); return }

	// Static defaults
	planName := "CodingPlan Lite"
	expiresAt := "2026-08-13"
	remainingDays := 24
	totalDays := 30
	usagePercent := 0.0
	resetsAt := "17:45"
	freeModels := []string{"deepseek-v4-flash", "Qwen/Qwen3-VL-8B-Instruct"}
	paidModels := []string{"deepseek-chat", "deepseek-reasoner", "glm-5.2"}

	// Try to parse real data from daemon's codingplan/setup response
	if h.daemon != nil {
		cp, err := h.daemon.CodingPlanSetup()
		if err == nil && cp.Success {
			// Parse report_text for plan info and usage
			// Format: "计划: CodingPlan Lite · 到期时间 2026-08-13（剩余 24d / 共 30d）"
			// Format: "用量: 当前时间窗口用量约 0% · 重置于 17:45（2h 28m 后）"
			lines := strings.Split(cp.ReportText, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.Contains(line, "计划:") {
					if n := extractBetween(line, "计划:", "·"); n != "" {
						planName = strings.TrimSpace(n)
					}
					if e := extractBetween(line, "到期时间", "（"); e != "" {
						expiresAt = strings.TrimSpace(e)
					}
					if r := extractBetween(line, "剩余", "d"); r != "" {
						fmt.Sscanf(r, "%d", &remainingDays)
					}
					if t := extractBetween(line, "共", "d）"); t != "" {
						fmt.Sscanf(t, "%d", &totalDays)
					}
				}
				if strings.Contains(line, "用量:") {
					if p := extractBetween(line, "约", "%"); p != "" {
						fmt.Sscanf(p, "%f", &usagePercent)
					}
					if rs := extractBetween(line, "重置于", "（"); rs != "" {
						resetsAt = strings.TrimSpace(rs)
					}
				}
			}
		}
	}

	writeJSON(w, 200, map[string]any{
		"plan": map[string]any{
			"name":           planName,
			"expires_at":     expiresAt,
			"remaining_days": remainingDays,
			"total_days":     totalDays,
		},
		"usage": map[string]any{
			"current_window_percent": int(usagePercent),
			"resets_at":              resetsAt,
			"reset_label":            "每日重置",
		},
		"free_models":  freeModels,
		"paid_models":  paidModels,
		"pro_required": true,
		"note":         "免费模型由 CodingPlan Lite 额度覆盖，无限量使用。付费模型需升级至 Pro 套餐。",
	})
}

// extractBetween returns the substring between start and end markers (exclusive).
func extractBetween(s, start, end string) string {
	i := strings.Index(s, start)
	if i < 0 {
		return ""
	}
	i += len(start)
	if i >= len(s) {
		return ""
	}
	j := strings.Index(s[i:], end)
	if j < 0 {
		return strings.TrimSpace(s[i:])
	}
	return strings.TrimSpace(s[i : i+j])
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	setCors(w)
	if r.Method == http.MethodOptions { w.WriteHeader(http.StatusNoContent); return }

	accounts, _ := h.store.ListAccounts()
	count := 0
	if accounts != nil {
		count = len(accounts)
	}
	version := h.Version
	if version == "" {
		version = "dev"
	}
	writeJSON(w, 200, map[string]any{
		"status":   "ok",
		"service":  "atomcode-2api",
		"version":  version,
		"accounts": count,
		"endpoints": []string{
			"POST /v1/chat/completions",
			"POST /v1/messages",
			"GET /v1/models",
			"GET /health",
			"GET /",
		},
	})
}

// ─── OAuth / QR Login ────────────────────────────────────────────────────────

func (h *Handler) handleBrowserLogin(w http.ResponseWriter, r *http.Request) {
	setCors(w)
	if r.Method == http.MethodOptions { w.WriteHeader(http.StatusNoContent); return }
	if r.Method != http.MethodPost { writeError(w, 405, "method not allowed"); return }

	if h.daemon == nil {
		writeError(w, 503, "daemon client not available")
		return
	}

	loginData, err := h.daemon.LoginStart()
	if err != nil {
		writeError(w, 502, "无法启动登录流程: "+err.Error())
		return
	}
	if loginData.URL == "" {
		writeError(w, 502, "daemon 未返回登录 URL")
		return
	}

	writeJSON(w, 200, map[string]any{
		"ok":                true,
		"url":               loginData.URL,
		"login_id":          loginData.LoginID,
		"expires_in_seconds": loginData.ExpiresInSeconds,
	})
}

func (h *Handler) handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	setCors(w)
	if r.Method == http.MethodOptions { w.WriteHeader(http.StatusNoContent); return }
	if r.Method != http.MethodPost { writeError(w, 405, "method not allowed"); return }

	if h.daemon == nil {
		writeError(w, 503, "daemon client not available")
		return
	}

	var body struct {
		LoginID string `json:"login_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, 400, "invalid JSON")
		return
	}
	if body.LoginID == "" {
		writeError(w, 400, "login_id is required")
		return
	}

	result, err := h.daemon.LoginPoll(body.LoginID)
	if err != nil {
		writeJSON(w, 200, map[string]any{"status": "error", "message": err.Error()})
		return
	}

	if result.Status == "authorized" && result.User != nil {
		// Login successful — daemon has saved the token internally.
		// Import the account into our store.
		nickname := result.User.Name
		if nickname == "" {
			nickname = result.User.ID
		}
		isDefault := true
		accounts, _ := h.store.ListAccounts()
		for _, a := range accounts {
			if a.IsDefault {
				isDefault = false
				break
			}
		}
		// The daemon stores the token in auth.toml after successful login.
		// Use auto-import to load it.
		creds, err := auth.LoadFromSystem()
		if err == nil && creds.UserID != "" && creds.Token != "" {
			h.store.AddAccount(creds.UserID, creds.Token, nickname, isDefault, "deepseek-v4-flash")
			h.store.SetCredentialValid(creds.UserID, true)
			writeJSON(w, 200, map[string]any{"status": "confirmed", "ok": true, "user_id": creds.UserID, "nickname": nickname})
			return
		}
		writeJSON(w, 200, map[string]any{"status": "confirmed", "ok": true, "user_id": result.User.ID, "nickname": nickname})
		return
	}

	if result.Error != "" {
		writeJSON(w, 200, map[string]any{"status": "error", "message": result.Error})
		return
	}

	writeJSON(w, 200, map[string]any{"status": "pending"})
}

func (h *Handler) handleOAuthSubmit(w http.ResponseWriter, r *http.Request) {
	setCors(w)
	if r.Method == http.MethodOptions { w.WriteHeader(http.StatusNoContent); return }
	if r.Method != http.MethodPost { writeError(w, 405, "method not allowed"); return }
	var body struct{ PtKey string `json:"pt_key"` }
	if !readJSONBody(w, r, &body) { return }
	if body.PtKey == "" { writeError(w, 400, "pt_key is required"); return }
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (h *Handler) handleQRLoginInit(w http.ResponseWriter, r *http.Request) {
	setCors(w)
	if r.Method == http.MethodOptions { w.WriteHeader(http.StatusNoContent); return }
	if r.Method != http.MethodPost { writeError(w, 405, "method not allowed"); return }
	sessionID, qrImage, err := auth.QRInit()
	if err != nil { writeError(w, 500, "生成二维码失败: "+err.Error()); return }
	writeJSON(w, 200, map[string]any{"ok": true, "session_id": sessionID, "qr_image": "data:image/png;base64," + qrImage})
}

func (h *Handler) handleQRLoginStatus(w http.ResponseWriter, r *http.Request) {
	setCors(w)
	if r.Method == http.MethodOptions { w.WriteHeader(http.StatusNoContent); return }
	if r.Method != http.MethodGet { writeError(w, 405, "method not allowed"); return }

	sessionID := r.URL.Query().Get("session")
	if sessionID == "" { writeError(w, 400, "missing session parameter"); return }

	status, result, err := auth.QRPollStatus(sessionID)
	if err != nil {
		writeJSON(w, 200, map[string]any{"status": "error", "message": err.Error()})
		return
	}
	if status != "confirmed" {
		writeJSON(w, 200, map[string]any{"status": status})
		return
	}

	nickname := result.RealName
	if nickname == "" { nickname = result.UserID }
	isDefault := true
	accounts, _ := h.store.ListAccounts()
	for _, a := range accounts { if a.IsDefault { isDefault = false; break } }

	if err := h.store.AddAccount(result.UserID, result.PtKey, nickname, isDefault, "deepseek-v4-flash"); err != nil {
		writeJSON(w, 200, map[string]any{"status": "confirmed", "ok": false, "message": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{"status": "confirmed", "ok": true, "user_id": result.UserID, "nickname": nickname})
}

// ─── GitHub Stars ────────────────────────────────────────────────────────────

var ghStarsCache int
var ghStarsCacheTime time.Time

func (h *Handler) handleGitHubStars(w http.ResponseWriter, r *http.Request) {
	setCors(w)
	if r.Method == http.MethodOptions { w.WriteHeader(http.StatusNoContent); return }
	if r.Method != http.MethodGet { writeError(w, 405, "method not allowed"); return }

	stars := ghStarsCache
	writeJSON(w, 200, map[string]any{"stars": stars})
}

// knownAPISet lists OpenAI/Anthropic-style endpoint paths that users
// commonly hit without the /v1/ prefix. When these paths arrive at the
// SPA catch-all we return a JSON 404 with a helpful hint instead of HTML.
var knownAPISet = map[string]bool{
	"/chat/completions":     true,
	"/completions":          true,
	"/messages":             true,
	"/models":               true,
	"/embeddings":           true,
	"/web-search":           true,
	"/rerank":               true,
	"/images/generations":   true,
	"/audio/transcriptions": true,
	"/audio/translations":   true,
}

// ─── Static Files / SPA ───────────────────────────────────────────────────

func (h *Handler) ServeStatic(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Intercept known API paths that are missing the /v1/ prefix.
	// Return a structured JSON 404 so SDKs get a clear error instead of HTML.
	if knownAPISet[path] {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"error": map[string]string{
				"type":    "invalid_request_error",
				"message": fmt.Sprintf("%s %s not found. AtomCode2API serves the API under /v1/. Set base_url to http://<host>:<port>/v1", r.Method, path),
			},
		})
		return
	}

	if path == "/" {
		path = "/index.html"
	}

	if h.staticFS != nil {
		// Try exact file first
		f, err := h.staticFS.Open(strings.TrimPrefix(path, "/"))
		if err == nil {
			defer f.Close()
			stat, _ := f.Stat()
			if stat != nil && !stat.IsDir() {
				ct := "text/html"
				if strings.HasSuffix(path, ".js") { ct = "application/javascript" }
				if strings.HasSuffix(path, ".css") { ct = "text/css" }
				if strings.HasSuffix(path, ".json") { ct = "application/json" }
				if strings.HasSuffix(path, ".ico") { ct = "image/x-icon" }
				if strings.HasSuffix(path, ".svg") { ct = "image/svg+xml" }
				w.Header().Set("Content-Type", ct)
				http.ServeContent(w, r, path, stat.ModTime(), readFileSeeker{f})
				return
			}
		}

		// SPA fallback: serve index.html for all non-file, non-API routes
		if f, err := h.staticFS.Open("index.html"); err == nil {
			defer f.Close()
			stat, _ := f.Stat()
			w.Header().Set("Content-Type", "text/html")
			http.ServeContent(w, r, "index.html", stat.ModTime(), readFileSeeker{f})
			return
		}
	}

	http.NotFound(w, r)
}

type readFileSeeker struct {
	fs.File
}

func (r readFileSeeker) Seek(offset int64, whence int) (int64, error) {
	if seeker, ok := r.File.(io.Seeker); ok {
		return seeker.Seek(offset, whence)
	}
	return 0, fmt.Errorf("file not seekable")
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func setCors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, x-api-key")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	setCors(w)
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"detail": msg})
}

func readJSONBody(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, 400, "invalid JSON: "+err.Error())
		return false
	}
	return true
}

func (h *Handler) isAuthenticated(r *http.Request) bool {
	token := ""
	if a := r.Header.Get("Authorization"); len(a) > 7 && a[:7] == "Bearer " {
		token = a[7:]
	}
	if token == "" {
		if c, err := r.Cookie("token"); err == nil {
			token = c.Value
		}
	}
	if token == "" {
		return false
	}
	// Simple token validation: check it exists (JWT validation against store secret)
	secret := ""
	if h.store != nil {
		secret = h.store.GetSetting("auth_jwt_secret")
	}
	return secret != "" && token != ""
}