package atmc

// --- Auth ---

type LoginStartRequest struct {
	OpenBrowser bool `json:"open_browser"`
}

type LoginStartResponse struct {
	LoginID          string `json:"login_id"`
	URL              string `json:"url"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}

type LoginPollResponse struct {
	Status string   `json:"status,omitempty"`
	User   *UserInfo `json:"user,omitempty"`
	Error  string   `json:"error,omitempty"`
}

type AuthStatusResponse struct {
	LoggedIn bool      `json:"logged_in"`
	User     *UserInfo `json:"user,omitempty"`
	Token    *TokenInfo `json:"token,omitempty"`
}

type UserInfo struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

type TokenInfo struct {
	AccessToken     string `json:"access_token,omitempty"`
	TokenType       string `json:"token_type,omitempty"`
	ExpiresIn       int    `json:"expires_in"`
	HasRefreshToken bool   `json:"has_refresh_token"`
}

// --- Chat ---

type ChatRequest struct {
	Message   string `json:"message"`
	Stream    bool   `json:"stream"`
	Provider  string `json:"provider,omitempty"`
	System    string `json:"system,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

type SSEEvent struct {
	Type      string `json:"type"`
	Content   string `json:"content,omitempty"`
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
	Chunk     string `json:"chunk,omitempty"`
	Output    string `json:"output,omitempty"`
	Success   *bool  `json:"success,omitempty"`
	DurationMs int    `json:"duration_ms,omitempty"`
	Prompt    int    `json:"prompt,omitempty"`
	Completion int   `json:"completion,omitempty"`
	Total     int    `json:"total,omitempty"`
	Message   string `json:"message,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	Provider  string `json:"provider,omitempty"`
}

// --- Daemon health ---

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version,omitempty"`
	Service string `json:"service,omitempty"`
}

// --- Models ---

type ModelInfo struct {
	ID      string `json:"id,omitempty"`
	Model   string `json:"model"`
	Name    string `json:"name,omitempty"`
	Provider string `json:"provider,omitempty"`
}

type ProviderConfig struct {
	Name            string `json:"name"`
	Model           string `json:"model"`
	BaseURL         string `json:"base_url"`
	ProviderType    string `json:"provider_type"`
	IsDefault       bool   `json:"is_default"`
}

// --- CodingPlan ---

type CodingPlanSetupResponse struct {
	Success         bool              `json:"success"`
	ReportText      string            `json:"report_text,omitempty"`
	DefaultProvider string            `json:"default_provider,omitempty"`
	Providers       []ProviderConfig  `json:"providers,omitempty"`
	Steps           map[string]any    `json:"steps,omitempty"`
}
