package store

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

const (
	DefaultDBDir  = ".atomcode-proxy"
	DefaultDBName = "proxy.db"
)

type Account struct {
	UserID       string `json:"user_id"`
	Nickname     string `json:"nickname"`
	Remark       string `json:"remark"`
	APIToken     string `json:"api_token"`
	IsDefault    bool   `json:"is_default"`
	DefaultModel string `json:"default_model"`
	CreatedAt    string `json:"created_at,omitempty"`
}

type RequestLog struct {
	ID           int64  `json:"id"`
	APIKey       string `json:"api_key"`
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

type Store struct {
	db     *sql.DB
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
			is_default INTEGER DEFAULT 0,
			default_model TEXT DEFAULT '',
			created_at TEXT DEFAULT (datetime('now', 'localtime')),
			updated_at TEXT DEFAULT (datetime('now', 'localtime'))
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

// ─── Accounts ─────────────────────────────────────────────────────────────────

func (s *Store) ListAccounts() ([]Account, error) {
	rows, err := s.db.Query("SELECT user_id, nickname, remark, api_token, is_default, default_model, created_at FROM accounts ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []Account
	for rows.Next() {
		var a Account
		var isDef int
		if err := rows.Scan(&a.UserID, &a.Nickname, &a.Remark, &a.APIToken, &isDef, &a.DefaultModel, &a.CreatedAt); err != nil {
			return nil, err
		}
		a.IsDefault = isDef == 1
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}

// ─── Settings ────────────────────────────────────────────────────────────────

func (s *Store) GetSetting(key string) string {
	var val string
	s.db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&val)
	return val
}

func (s *Store) SetSetting(key, value string) error {
	_, err := s.db.Exec(
		"INSERT OR REPLACE INTO settings (key, value, updated_at) VALUES (?, ?, datetime('now', 'localtime'))",
		key, value,
	)
	return err
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

func (s *Store) CleanupOldLogs(days int) {
	if days <= 0 {
		return
	}
	result, err := s.db.Exec(
		"DELETE FROM request_logs WHERE created_at < datetime('now', '-' || ? || ' days')",
		days,
	)
	if err == nil {
		if n, _ := result.RowsAffected(); n > 0 {
			log.Printf("store: cleaned up %d old log entries", n)
		}
	}
}
