package store

import (
	"net/http"
	"sync"
)

// ─── Context key for token usage tracking ──────────────────────────────────

type contextKey string

const (
	tokenUsageKey  contextKey = "token_usage"
	modelKey       contextKey = "request_model"
	accountModelKey contextKey = "account_default_model"
)

// TokenUsage holds input/output token counts for a request.
type TokenUsage struct {
	InputTokens  int
	OutputTokens int
	mu           sync.Mutex
}

// InitTokenUsage initializes token usage tracking on a request context.
func InitTokenUsage(r *http.Request) *http.Request {
	SetTokenUsage(r, 0, 0)
	return r
}

// SetTokenUsage stores token usage in the request.
func SetTokenUsage(r *http.Request, in, out int) {
	r.Header.Set("X-Token-Usage-In", intToStr(in))
	r.Header.Set("X-Token-Usage-Out", intToStr(out))
}

// GetTokenUsage retrieves token usage from the request.
func GetTokenUsage(r *http.Request) (int, int) {
	in := parseStrToInt(r.Header.Get("X-Token-Usage-In"))
	out := parseStrToInt(r.Header.Get("X-Token-Usage-Out"))
	return in, out
}

// InitModel initializes model tracking.
func InitModel(r *http.Request) *http.Request {
	SetModel(r, "")
	return r
}

// SetModel stores the model name in the request.
func SetModel(r *http.Request, model string) {
	r.Header.Set("X-Request-Model", model)
}

// GetModel retrieves the model name from the request.
func GetModel(r *http.Request) string {
	return r.Header.Get("X-Request-Model")
}

// InitAccountModel initializes account default model tracking.
func InitAccountModel(r *http.Request) *http.Request {
	SetAccountDefaultModel(r, "")
	return r
}

// SetAccountDefaultModel stores the account default model.
func SetAccountDefaultModel(r *http.Request, model string) {
	r.Header.Set("X-Account-Default-Model", model)
}

// GetAccountDefaultModel retrieves the account default model.
func GetAccountDefaultModel(r *http.Request) string {
	return r.Header.Get("X-Account-Default-Model")
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}

func parseStrToInt(s string) int {
	if s == "" {
		return 0
	}
	n := 0
	neg := false
	for i, c := range s {
		if i == 0 && c == '-' {
			neg = true
			continue
		}
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			return 0
		}
	}
	if neg {
		return -n
	}
	return n
}