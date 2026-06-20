package store

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	DefaultDBDir  = ".atomcode-proxy"
	DefaultDBName = "proxy.db"
	encKeyFile    = ".enc_key"
	MaxAccounts   = 10
)

type Account struct {
	UserID       string `json:"user_id"`
	Nickname     string `json:"nickname"`
	Remark       string `json:"remark"`
	APIToken     string `json:"api_token"`
	PtKey        string `json:"-"`
	IsDefault    bool   `json:"is_default"`
	DefaultModel string `json:"default_model"`
	CreatedAt    string `json:"created_at,omitempty"`
}

func (a *Account) DisplayName() string {
	if a.Remark != "" {
		return a.Remark
	}
	if a.Nickname != "" {
		return a.Nickname
	}
	return a.UserID
}

type AccountInfo struct {
	UserID          string `json:"user_id"`
	Nickname        string `json:"nickname"`
	Remark          string `json:"remark"`
	APIToken        string `json:"api_token"`
	IsDefault       bool   `json:"is_default"`
	DefaultModel    string `json:"default_model"`
	CreatedAt       string `json:"created_at,omitempty"`
	TotalRequests   int    `json:"total_requests"`
	TodayRequests   int    `json:"today_requests"`
	TotalTokens     int    `json:"total_tokens"`
	TodayTokens     int    `json:"today_tokens"`
	CredentialValid int    `json:"credential_valid"`
	CredentialCheckedAt string `json:"credential_checked_at,omitempty"`
	CredentialError string `json:"credential_error,omitempty"`
}

func (a *AccountInfo) DisplayName() string {
	if a.Remark != "" {
		return a.Remark
	}
	if a.Nickname != "" {
		return a.Nickname
	}
	return a.UserID
}

type Stats struct {
	TotalRequests  int            `json:"total_requests"`
	TotalInputTk   int            `json:"total_input_tokens"`
	TotalOutputTk  int            `json:"total_output_tokens"`
	AccountsCount  int            `json:"accounts_count"`
	AvgLatencyMs   float64        `json:"avg_latency_ms"`
	ErrorCount     int            `json:"error_count"`
	StreamCount    int            `json:"stream_count"`
	SuccessCount   int            `json:"success_count"`
	ByModel        []ModelCount   `json:"by_model"`
	ByAccount      []AccountCount `json:"by_account"`
}

type ModelCount struct {
	Model string `json:"model"`
	Count int    `json:"count"`
}

type AccountCount struct {
	UserID   string `json:"user_id"`
	Nickname string `json:"nickname"`
	Count    int    `json:"count"`
}

func (a *AccountCount) DisplayName() string {
	if a.Nickname != "" {
		return a.Nickname
	}
	return a.UserID
}

type RequestLog struct {
	ID           int64  `json:"id"`
	APIKey       string `json:"-"`
	Model        string `json:"model"`
	Endpoint     string `json:"endpoint"`
	Stream       bool   `json:"stream"`
	StatusCode   int    `json:"status_code"`
	LatencyMs    int64  `json:"latency_ms"`
	ErrorMessage string `json:"error_message"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
	CreatedAt    string `json:"created_at"`
}

type AllTimeTotals struct {
	TotalRequests int `json:"total_requests"`
	TotalInputTk  int `json:"total_input_tokens"`
	TotalOutputTk int `json:"total_output_tokens"`
	ErrorCount    int `json:"error_count"`
}

type HourlyData struct {
	Hour        string `json:"hour"`
	Count       int    `json:"count"`
	InputTokens int    `json:"input_tokens"`
	OutputTokens int   `json:"output_tokens"`
	Errors      int    `json:"errors"`
}

type AccountStats struct {
	UserID        string          `json:"user_id"`
	TotalRequests int             `json:"total_requests"`
	TotalInputTk  int             `json:"total_input_tokens"`
	TotalOutputTk int             `json:"total_output_tokens"`
	SuccessCount  int             `json:"success_count"`
	StreamCount   int             `json:"stream_count"`
	AvgLatencyMs  float64         `json:"avg_latency_ms"`
	ErrorCount    int             `json:"error_count"`
	ByModel       []ModelCount    `json:"by_model"`
	ByEndpoint    []EndpointCount `json:"by_endpoint"`
	AllTime       *AllTimeTotals  `json:"all_time"`
	Hourly        []HourlyData    `json:"hourly"`
}

type EndpointCount struct {
	Endpoint string `json:"endpoint"`
	Count    int    `json:"count"`
}

type ExportAccountItem struct {
	UserID       string `json:"user_id"`
	Nickname     string `json:"nickname"`
	Remark       string `json:"remark"`
	PtKey        string `json:"pt_key"`
	IsDefault    bool   `json:"is_default"`
	DefaultModel string `json:"default_model"`
}

type Store struct {
	db     *sql.DB
	enc    cipher.AEAD
	mu     sync.Mutex
	dbPath string
}

func DefaultDBPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, DefaultDBDir)
	return filepath.Join(dir, DefaultDBName), nil
}

func Open(dbPath string) (*Store, error) {
	if dbPath == "" {
		var err error
		dbPath, err = DefaultDBPath()
		if err != nil {
			return nil, err
		}
	}

	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	s := &Store{db: db, dbPath: dbPath}

	encKey, err := s.loadOrCreateEncKey(dir)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("encryption key: %w", err)
	}

	block, err := aes.NewCipher(encKey)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	s.enc, err = cipher.NewGCM(block)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS accounts (
			user_id TEXT PRIMARY KEY,
			nickname TEXT DEFAULT '',
			remark TEXT DEFAULT '',
			api_token TEXT NOT NULL DEFAULT '',
			pt_key TEXT NOT NULL DEFAULT '',
			is_default INTEGER DEFAULT 0,
			default_model TEXT DEFAULT '',
			created_at TEXT DEFAULT (datetime('now', 'localtime')),
			updated_at TEXT DEFAULT (datetime('now', 'localtime')),
			credential_refreshed_at TEXT DEFAULT '',
			credential_valid INTEGER DEFAULT -1
		);
		CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TEXT DEFAULT (datetime('now', 'localtime'))
		);
		CREATE TABLE IF NOT EXISTS request_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			api_key TEXT,
			model TEXT,
			endpoint TEXT,
			stream INTEGER DEFAULT 0,
			status_code INTEGER,
			latency_ms INTEGER,
			error_message TEXT DEFAULT '',
			input_tokens INTEGER DEFAULT 0,
			output_tokens INTEGER DEFAULT 0,
			created_at TEXT DEFAULT (datetime('now', 'localtime'))
		);
	`)
	return err
}

func (s *Store) loadOrCreateEncKey(dir string) ([]byte, error) {
	keyPath := filepath.Join(dir, encKeyFile)
	data, err := os.ReadFile(keyPath)
	if err == nil {
		key, err := hex.DecodeString(string(data))
		if err == nil && len(key) == 32 {
			return key, nil
		}
	}
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}
	if err := os.WriteFile(keyPath, []byte(hex.EncodeToString(key)), 0600); err != nil {
		return nil, fmt.Errorf("write key: %w", err)
	}
	return key, nil
}

func (s *Store) encrypt(plaintext string) (string, error) {
	nonce := make([]byte, s.enc.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := s.enc.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
}

func (s *Store) decrypt(ciphertext string) (string, error) {
	data, err := hex.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	nonceSize := s.enc.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ct := data[:nonceSize], data[nonceSize:]
	plaintext, err := s.enc.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func generateToken() string {
	b := make([]byte, 32)
	io.ReadFull(rand.Reader, b)
	return "sk-atmc-" + hex.EncodeToString(b)
}

// ─── Accounts ─────────────────────────────────────────────────────────────────

func (s *Store) AddAccount(userID, ptKey, nickname string, isDefault bool, defaultModel string) error {
	if userID == "" {
		return fmt.Errorf("user_id cannot be empty")
	}
	if ptKey == "" {
		return fmt.Errorf("pt_key cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var existingToken string
	err := s.db.QueryRow("SELECT api_token FROM accounts WHERE user_id = ?", userID).Scan(&existingToken)
	if err == nil {
		encPtKey, err := s.encrypt(ptKey)
		if err != nil {
			return fmt.Errorf("encrypt pt_key: %w", err)
		}
		_, err = s.db.Exec(
			"UPDATE accounts SET pt_key = ?, nickname = CASE WHEN nickname = '' OR nickname IS NULL THEN ? ELSE nickname END, updated_at = datetime('now', 'localtime') WHERE user_id = ?",
			encPtKey, nickname, userID,
		)
		return err
	}

	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM accounts").Scan(&count)
	if count >= MaxAccounts {
		return fmt.Errorf("账号数量已达上限（%d 个）", MaxAccounts)
	}

	encPtKey, err := s.encrypt(ptKey)
	if err != nil {
		return fmt.Errorf("encrypt pt_key: %w", err)
	}

	if isDefault {
		s.db.Exec("UPDATE accounts SET is_default = 0 WHERE is_default = 1")
	}

	def := 0
	if isDefault {
		def = 1
	}

	token := generateToken()
	_, err = s.db.Exec(
		"INSERT INTO accounts (user_id, nickname, api_token, pt_key, is_default, default_model) VALUES (?, ?, ?, ?, ?, ?)",
		userID, nickname, token, encPtKey, def, defaultModel,
	)
	return err
}

func (s *Store) ListAccounts() ([]AccountInfo, error) {
	rows, err := s.db.Query("SELECT user_id, nickname, remark, api_token, is_default, default_model, created_at, credential_valid, credential_refreshed_at FROM accounts ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	accounts := make([]AccountInfo, 0)
	for rows.Next() {
		var a AccountInfo
		var isDef int
		if err := rows.Scan(&a.UserID, &a.Nickname, &a.Remark, &a.APIToken, &isDef, &a.DefaultModel, &a.CreatedAt, &a.CredentialValid, &a.CredentialCheckedAt); err != nil {
			return nil, err
		}
		a.IsDefault = isDef == 1
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}

func (s *Store) FillAccountStats(accounts []AccountInfo) {
	if len(accounts) == 0 {
		return
	}
	allRows, err := s.db.Query(`
		SELECT api_key, COUNT(*) as req_count, COALESCE(SUM(input_tokens + output_tokens), 0) as token_sum
		FROM request_logs GROUP BY api_key`)
	if err != nil {
		return
	}
	allMap := make(map[string][2]int)
	for allRows.Next() {
		var key string
		var reqCount, tokenSum int
		if allRows.Scan(&key, &reqCount, &tokenSum) == nil {
			allMap[key] = [2]int{reqCount, tokenSum}
		}
	}
	allRows.Close()

	todayRows, err := s.db.Query(`
		SELECT api_key, COUNT(*) as req_count, COALESCE(SUM(input_tokens + output_tokens), 0) as token_sum
		FROM request_logs WHERE date(created_at, 'localtime') = date('now', 'localtime') GROUP BY api_key`)
	if err != nil {
		return
	}
	todayMap := make(map[string][2]int)
	for todayRows.Next() {
		var key string
		var reqCount, tokenSum int
		if todayRows.Scan(&key, &reqCount, &tokenSum) == nil {
			todayMap[key] = [2]int{reqCount, tokenSum}
		}
	}
	todayRows.Close()

	for i := range accounts {
		if v, ok := allMap[accounts[i].UserID]; ok {
			accounts[i].TotalRequests = v[0]
			accounts[i].TotalTokens = v[1]
		}
		if v, ok := todayMap[accounts[i].UserID]; ok {
			accounts[i].TodayRequests = v[0]
			accounts[i].TodayTokens = v[1]
		}
	}
}

func (s *Store) GetAccount(userID string) (*Account, error) {
	var a Account
	var encPtKey string
	var isDef int
	err := s.db.QueryRow(
		"SELECT user_id, nickname, remark, api_token, pt_key, is_default, default_model, created_at FROM accounts WHERE user_id = ?",
		userID,
	).Scan(&a.UserID, &a.Nickname, &a.Remark, &a.APIToken, &encPtKey, &isDef, &a.DefaultModel, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	ptKey, err := s.decrypt(encPtKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	a.PtKey = ptKey
	a.IsDefault = isDef == 1
	return &a, nil
}

func (s *Store) GetAccountByToken(token string) (*Account, error) {
	var a Account
	var encPtKey string
	var isDef int
	err := s.db.QueryRow(
		"SELECT user_id, nickname, remark, api_token, pt_key, is_default, default_model, created_at FROM accounts WHERE api_token = ?",
		token,
	).Scan(&a.UserID, &a.Nickname, &a.Remark, &a.APIToken, &encPtKey, &isDef, &a.DefaultModel, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	ptKey, err := s.decrypt(encPtKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	a.PtKey = ptKey
	a.IsDefault = isDef == 1
	return &a, nil
}

func (s *Store) GetDefaultAccount() (*Account, error) {
	var a Account
	var encPtKey string
	err := s.db.QueryRow(
		"SELECT user_id, nickname, remark, api_token, pt_key, is_default, default_model, created_at FROM accounts WHERE is_default = 1 LIMIT 1",
	).Scan(&a.UserID, &a.Nickname, &a.Remark, &a.APIToken, &encPtKey, new(int), &a.DefaultModel, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	ptKey, err := s.decrypt(encPtKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	a.PtKey = ptKey
	a.IsDefault = true
	return &a, nil
}

func (s *Store) RenewToken(userID string) (string, error) {
	token := generateToken()
	_, err := s.db.Exec("UPDATE accounts SET api_token = ?, updated_at = datetime('now', 'localtime') WHERE user_id = ?", token, userID)
	if err != nil {
		return "", err
	}
	return token, nil
}

func (s *Store) SetDefault(userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("UPDATE accounts SET is_default = 0, updated_at = datetime('now', 'localtime')"); err != nil {
		return err
	}
	if _, err := tx.Exec("UPDATE accounts SET is_default = 1, updated_at = datetime('now', 'localtime') WHERE user_id = ?", userID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) RemoveAccount(userID string) error {
	_, err := s.db.Exec("DELETE FROM accounts WHERE user_id = ?", userID)
	return err
}

func (s *Store) ClearAllAccounts() (int, error) {
	result, err := s.db.Exec("DELETE FROM accounts")
	if err != nil {
		return 0, err
	}
	n, _ := result.RowsAffected()
	return int(n), nil
}

func (s *Store) UpdateAccountModel(userID, model string) error {
	_, err := s.db.Exec(
		"UPDATE accounts SET default_model = ?, updated_at = datetime('now', 'localtime') WHERE user_id = ?",
		model, userID,
	)
	return err
}

func (s *Store) UpdateRemark(userID, remark string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	result, err := s.db.Exec("UPDATE accounts SET remark = ?, updated_at = datetime('now', 'localtime') WHERE user_id = ?", remark, userID)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("account %q not found", userID)
	}
	return nil
}

func (s *Store) UpdatePtKey(userID, ptKey string) error {
	encPtKey, err := s.encrypt(ptKey)
	if err != nil {
		return fmt.Errorf("encrypt: %w", err)
	}
	result, err := s.db.Exec(
		"UPDATE accounts SET pt_key = ?, updated_at = datetime('now', 'localtime'), credential_refreshed_at = datetime('now', 'localtime') WHERE user_id = ?",
		encPtKey, userID,
	)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	slog.Info("store: pt_key updated", "user_id", userID, "rows_affected", rows)
	return nil
}

func (s *Store) SetCredentialValid(userID string, valid bool) {
	v := 0
	if valid {
		v = 1
	}
	s.db.Exec("UPDATE accounts SET credential_valid = ?, credential_refreshed_at = datetime('now', 'localtime') WHERE user_id = ?", v, userID)
}

func (s *Store) ListStaleAccounts(threshold time.Duration) ([]Account, error) {
	normalCutoff := time.Now().Add(-threshold).Format("2006-01-02 15:04:05")
	backoffCutoff := time.Now().Add(-threshold * 4).Format("2006-01-02 15:04:05")
	rows, err := s.db.Query(
		`SELECT user_id, nickname, pt_key, default_model FROM accounts
		 WHERE credential_refreshed_at = ''
		    OR credential_valid = -1
		    OR (credential_valid = 1 AND credential_refreshed_at < ?)
		    OR (credential_valid = 0 AND credential_refreshed_at < ?)
		 ORDER BY created_at`,
		normalCutoff, backoffCutoff,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	accounts := make([]Account, 0)
	for rows.Next() {
		var a Account
		var encPtKey string
		if err := rows.Scan(&a.UserID, &a.Nickname, &encPtKey, &a.DefaultModel); err != nil {
			continue
		}
		ptKey, err := s.decrypt(encPtKey)
		if err != nil {
			continue
		}
		a.PtKey = ptKey
		accounts = append(accounts, a)
	}
	return accounts, nil
}

func (s *Store) ListAllAccountsWithCredentials() ([]Account, error) {
	rows, err := s.db.Query("SELECT user_id, nickname, pt_key, default_model FROM accounts ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	accounts := make([]Account, 0)
	for rows.Next() {
		var a Account
		var encPtKey string
		if err := rows.Scan(&a.UserID, &a.Nickname, &encPtKey, &a.DefaultModel); err != nil {
			continue
		}
		ptKey, err := s.decrypt(encPtKey)
		if err != nil {
			continue
		}
		a.PtKey = ptKey
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}

// ─── Export / Import ─────────────────────────────────────────────────────────

func (s *Store) ExportAccounts() ([]ExportAccountItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query("SELECT user_id, nickname, remark, pt_key, is_default, default_model FROM accounts ORDER BY created_at")
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	items := make([]ExportAccountItem, 0)
	for rows.Next() {
		var item ExportAccountItem
		var encPtKey string
		var isDef int
		if err := rows.Scan(&item.UserID, &item.Nickname, &item.Remark, &encPtKey, &isDef, &item.DefaultModel); err != nil {
			return nil, err
		}
		ptKey, err := s.decrypt(encPtKey)
		if err != nil {
			continue
		}
		item.PtKey = ptKey
		item.IsDefault = isDef == 1
		items = append(items, item)
	}
	return items, nil
}

func (s *Store) ImportAccounts(items []ExportAccountItem) (added, updated int, err error) {
	for _, item := range items {
		if item.UserID == "" || item.PtKey == "" {
			continue
		}
		var existing int
		s.mu.Lock()
		e := s.db.QueryRow("SELECT COUNT(*) FROM accounts WHERE user_id = ?", item.UserID).Scan(&existing)
		s.mu.Unlock()
		if e != nil {
			return added, updated, e
		}
		if err := s.AddAccount(item.UserID, item.PtKey, item.Nickname, item.IsDefault, item.DefaultModel); err != nil {
			return added, updated, err
		}
		if existing > 0 {
			updated++
		} else {
			added++
		}
	}
	return added, updated, nil
}

// ─── Settings ────────────────────────────────────────────────────────────────

func (s *Store) GetSetting(key string) string {
	var val string
	s.db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&val)
	return val
}

func (s *Store) GetIntSetting(key string, defaultVal int) int {
	v := s.GetSetting(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}

func (s *Store) SetSetting(key, value string) error {
	_, err := s.db.Exec(
		"INSERT OR REPLACE INTO settings (key, value, updated_at) VALUES (?, ?, datetime('now', 'localtime'))",
		key, value,
	)
	return err
}

func (s *Store) SetSettings(settings map[string]string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for k, v := range settings {
		if _, err := tx.Exec(
			"INSERT OR REPLACE INTO settings (key, value, updated_at) VALUES (?, ?, datetime('now', 'localtime'))",
			k, v,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) GetSettings() (map[string]string, error) {
	rows, err := s.db.Query("SELECT key, value FROM settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		m[k] = v
	}
	return m, rows.Err()
}

// ─── Request Logging ─────────────────────────────────────────────────────────

func (s *Store) LogRequest(apiKey, model, endpoint string, stream bool, statusCode int, latencyMs int64, errMsg string, inputTokens, outputTokens int) error {
	sInt := 0
	if stream {
		sInt = 1
	}
	_, err := s.db.Exec(
		"INSERT INTO request_logs (api_key, model, endpoint, stream, status_code, latency_ms, error_message, input_tokens, output_tokens) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		apiKey, model, endpoint, sInt, statusCode, latencyMs, errMsg, inputTokens, outputTokens,
	)
	if err != nil {
		log.Printf("store: log request failed: %v", err)
	}
	return err
}

func (s *Store) CleanupOldLogs(days int) (int64, error) {
	if days <= 0 {
		return 0, nil
	}
	result, err := s.db.Exec(
		"DELETE FROM request_logs WHERE created_at < datetime('now', '-' || ? || ' days')",
		days,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// ─── Stats ────────────────────────────────────────────────────────────────────

func (s *Store) GetStats() (*Stats, error) {
	stats := &Stats{}
	tf := "date(created_at, 'localtime') = date('now', 'localtime')"

	s.db.QueryRow("SELECT COUNT(*) FROM request_logs WHERE "+tf).Scan(&stats.TotalRequests)
	s.db.QueryRow("SELECT COALESCE(AVG(latency_ms), 0) FROM request_logs WHERE "+tf).Scan(&stats.AvgLatencyMs)
	s.db.QueryRow("SELECT COUNT(*) FROM accounts").Scan(&stats.AccountsCount)
	s.db.QueryRow("SELECT COUNT(*) FROM request_logs WHERE "+tf+" AND status_code >= 400").Scan(&stats.ErrorCount)
	s.db.QueryRow("SELECT COUNT(*) FROM request_logs WHERE "+tf+" AND stream = 1").Scan(&stats.StreamCount)
	s.db.QueryRow("SELECT COUNT(*) FROM request_logs WHERE "+tf+" AND status_code < 400").Scan(&stats.SuccessCount)
	s.db.QueryRow("SELECT COALESCE(SUM(input_tokens), 0) FROM request_logs WHERE "+tf).Scan(&stats.TotalInputTk)
	s.db.QueryRow("SELECT COALESCE(SUM(output_tokens), 0) FROM request_logs WHERE "+tf).Scan(&stats.TotalOutputTk)

	if rows, err := s.db.Query("SELECT model, COUNT(*) as cnt FROM request_logs WHERE "+tf+" AND model != '' GROUP BY model ORDER BY cnt DESC"); err == nil {
		defer rows.Close()
		for rows.Next() {
			var mc ModelCount
			if rows.Scan(&mc.Model, &mc.Count) == nil {
				stats.ByModel = append(stats.ByModel, mc)
			}
		}
	}

	validKeys := make(map[string]bool)
	accounts, _ := s.ListAccounts()
	for _, a := range accounts {
		validKeys[a.UserID] = true
	}

	if rows2, err := s.db.Query("SELECT api_key, COUNT(*) as cnt FROM request_logs WHERE "+tf+" GROUP BY api_key ORDER BY cnt DESC"); err == nil {
		defer rows2.Close()
		otherCount := 0
		for rows2.Next() {
			var ac AccountCount
			if rows2.Scan(&ac.UserID, &ac.Count) == nil {
				if validKeys[ac.UserID] {
					stats.ByAccount = append(stats.ByAccount, ac)
				} else {
					otherCount += ac.Count
				}
			}
		}
		if otherCount > 0 {
			stats.ByAccount = append(stats.ByAccount, AccountCount{UserID: "其他", Count: otherCount})
		}
	}

	return stats, nil
}

func (s *Store) GetAllTimeTotals() (*AllTimeTotals, error) {
	t := &AllTimeTotals{}
	s.db.QueryRow("SELECT COUNT(*) FROM request_logs").Scan(&t.TotalRequests)
	s.db.QueryRow("SELECT COALESCE(SUM(input_tokens), 0) FROM request_logs").Scan(&t.TotalInputTk)
	s.db.QueryRow("SELECT COALESCE(SUM(output_tokens), 0) FROM request_logs").Scan(&t.TotalOutputTk)
	s.db.QueryRow("SELECT COUNT(*) FROM request_logs WHERE status_code >= 400").Scan(&t.ErrorCount)
	return t, nil
}

func (s *Store) GetHourlyStats() ([]HourlyData, error) {
	rows, err := s.db.Query(`
		SELECT strftime('%m-%d %H', created_at, 'localtime') as hour,
			COUNT(*) as count,
			COALESCE(SUM(input_tokens), 0),
			COALESCE(SUM(output_tokens), 0),
			SUM(CASE WHEN status_code >= 400 THEN 1 ELSE 0 END)
		FROM request_logs WHERE created_at >= datetime('now', '-24 hours')
		GROUP BY hour ORDER BY hour`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]HourlyData, 0)
	for rows.Next() {
		var h HourlyData
		if rows.Scan(&h.Hour, &h.Count, &h.InputTokens, &h.OutputTokens, &h.Errors) == nil {
			result = append(result, h)
		}
	}
	return result, rows.Err()
}

func (s *Store) GetAccountStats(userID string) (*AccountStats, error) {
	as := &AccountStats{UserID: userID}
	tf := "created_at >= datetime('now', '-24 hours')"

	s.db.QueryRow("SELECT COUNT(*) FROM request_logs WHERE api_key = ? AND "+tf, userID).Scan(&as.TotalRequests)
	s.db.QueryRow("SELECT COALESCE(AVG(latency_ms), 0) FROM request_logs WHERE api_key = ? AND "+tf, userID).Scan(&as.AvgLatencyMs)
	s.db.QueryRow("SELECT COUNT(*) FROM request_logs WHERE api_key = ? AND stream = 1 AND "+tf, userID).Scan(&as.StreamCount)
	s.db.QueryRow("SELECT COUNT(*) FROM request_logs WHERE api_key = ? AND status_code >= 400 AND "+tf, userID).Scan(&as.ErrorCount)
	s.db.QueryRow("SELECT COUNT(*) FROM request_logs WHERE api_key = ? AND status_code < 400 AND "+tf, userID).Scan(&as.SuccessCount)
	s.db.QueryRow("SELECT COALESCE(SUM(input_tokens), 0) FROM request_logs WHERE api_key = ? AND "+tf, userID).Scan(&as.TotalInputTk)
	s.db.QueryRow("SELECT COALESCE(SUM(output_tokens), 0) FROM request_logs WHERE api_key = ? AND "+tf, userID).Scan(&as.TotalOutputTk)

	if rows, err := s.db.Query("SELECT model, COUNT(*) as cnt FROM request_logs WHERE api_key = ? AND "+tf+" GROUP BY model ORDER BY cnt DESC", userID); err == nil {
		defer rows.Close()
		for rows.Next() {
			var mc ModelCount
			if rows.Scan(&mc.Model, &mc.Count) == nil {
				as.ByModel = append(as.ByModel, mc)
			}
		}
	}

	if rows2, err := s.db.Query("SELECT endpoint, COUNT(*) as cnt FROM request_logs WHERE api_key = ? AND "+tf+" GROUP BY endpoint ORDER BY cnt DESC", userID); err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var ec EndpointCount
			if rows2.Scan(&ec.Endpoint, &ec.Count) == nil {
				as.ByEndpoint = append(as.ByEndpoint, ec)
			}
		}
	}

	allTime := &AllTimeTotals{}
	s.db.QueryRow("SELECT COUNT(*) FROM request_logs WHERE api_key = ?", userID).Scan(&allTime.TotalRequests)
	s.db.QueryRow("SELECT COALESCE(SUM(input_tokens), 0) FROM request_logs WHERE api_key = ?", userID).Scan(&allTime.TotalInputTk)
	s.db.QueryRow("SELECT COALESCE(SUM(output_tokens), 0) FROM request_logs WHERE api_key = ?", userID).Scan(&allTime.TotalOutputTk)
	s.db.QueryRow("SELECT COUNT(*) FROM request_logs WHERE api_key = ? AND status_code >= 400", userID).Scan(&allTime.ErrorCount)
	as.AllTime = allTime

	if hRows, err := s.db.Query(`
		SELECT strftime('%m-%d %H', created_at, 'localtime') as hour,
			COUNT(*) as count,
			COALESCE(SUM(input_tokens), 0),
			COALESCE(SUM(output_tokens), 0),
			SUM(CASE WHEN status_code >= 400 THEN 1 ELSE 0 END)
		FROM request_logs WHERE api_key = ? AND `+tf+`
		GROUP BY hour ORDER BY hour`, userID); err == nil {
		defer hRows.Close()
		for hRows.Next() {
			var h HourlyData
			if hRows.Scan(&h.Hour, &h.Count, &h.InputTokens, &h.OutputTokens, &h.Errors) == nil {
				as.Hourly = append(as.Hourly, h)
			}
		}
	}

	return as, nil
}

func (s *Store) GetAccountLogs(userID string, limit int) ([]RequestLog, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.Query(
		"SELECT id, api_key, model, endpoint, stream, status_code, latency_ms, COALESCE(error_message, ''), COALESCE(input_tokens, 0), COALESCE(output_tokens, 0), created_at FROM request_logs WHERE api_key = ? ORDER BY id DESC LIMIT ?",
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := make([]RequestLog, 0)
	for rows.Next() {
		var l RequestLog
		var streamInt int
		if err := rows.Scan(&l.ID, &l.APIKey, &l.Model, &l.Endpoint, &streamInt, &l.StatusCode, &l.LatencyMs, &l.ErrorMessage, &l.InputTokens, &l.OutputTokens, &l.CreatedAt); err != nil {
			return nil, err
		}
		l.Stream = streamInt == 1
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

func (s *Store) GetRecentLogs(limit int) ([]RequestLog, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(
		"SELECT id, api_key, model, endpoint, stream, status_code, latency_ms, COALESCE(error_message, ''), COALESCE(input_tokens, 0), COALESCE(output_tokens, 0), created_at FROM request_logs ORDER BY id DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := make([]RequestLog, 0)
	for rows.Next() {
		var l RequestLog
		var streamInt int
		if err := rows.Scan(&l.ID, &l.APIKey, &l.Model, &l.Endpoint, &streamInt, &l.StatusCode, &l.LatencyMs, &l.ErrorMessage, &l.InputTokens, &l.OutputTokens, &l.CreatedAt); err != nil {
			return nil, err
		}
		l.Stream = streamInt == 1
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

func (s *Store) GetRecentErrors(limit int) ([]RequestLog, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(
		"SELECT id, api_key, model, endpoint, stream, status_code, latency_ms, COALESCE(error_message, ''), COALESCE(input_tokens, 0), COALESCE(output_tokens, 0), created_at FROM request_logs WHERE status_code >= 400 ORDER BY id DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := make([]RequestLog, 0)
	for rows.Next() {
		var l RequestLog
		var streamInt int
		if err := rows.Scan(&l.ID, &l.APIKey, &l.Model, &l.Endpoint, &streamInt, &l.StatusCode, &l.LatencyMs, &l.ErrorMessage, &l.InputTokens, &l.OutputTokens, &l.CreatedAt); err != nil {
			return nil, err
		}
		l.Stream = streamInt == 1
		logs = append(logs, l)
	}
	return logs, nil
}

// ─── Data Dir ────────────────────────────────────────────────────────────────

func EnsureDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, DefaultDBDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

func DBExists() bool {
	path, err := DefaultDBPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}
