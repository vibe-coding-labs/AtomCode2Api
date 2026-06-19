package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenAndClose(t *testing.T) {
	dir, _ := os.MkdirTemp("", "store-test")
	defer os.RemoveAll(dir)
	dbPath := filepath.Join(dir, "test.db")

	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()

	if s == nil {
		t.Fatal("store is nil")
	}
}

func TestSettings(t *testing.T) {
	dir, _ := os.MkdirTemp("", "store-test")
	defer os.RemoveAll(dir)

	s, _ := Open(filepath.Join(dir, "test.db"))
	defer s.Close()

	if err := s.SetSetting("foo", "bar"); err != nil {
		t.Fatalf("set: %v", err)
	}
	if v := s.GetSetting("foo"); v != "bar" {
		t.Errorf("expected bar, got %s", v)
	}
	if v := s.GetSetting("nonexistent"); v != "" {
		t.Errorf("expected empty, got %s", v)
	}
}

func TestSettingsMap(t *testing.T) {
	dir, _ := os.MkdirTemp("", "store-test")
	defer os.RemoveAll(dir)

	s, _ := Open(filepath.Join(dir, "test.db"))
	defer s.Close()

	s.SetSetting("a", "1")
	s.SetSetting("b", "2")
	m, err := s.GetSettings()
	if err != nil {
		t.Fatalf("get settings: %v", err)
	}
	if m["a"] != "1" || m["b"] != "2" {
		t.Errorf("unexpected settings: %v", m)
	}
}

func TestLogRequest(t *testing.T) {
	dir, _ := os.MkdirTemp("", "store-test")
	defer os.RemoveAll(dir)

	s, _ := Open(filepath.Join(dir, "test.db"))
	defer s.Close()

	if err := s.LogRequest("key1", "model1", "/v1/chat", true, 200, 100, "", 10, 20); err != nil {
		t.Fatalf("log: %v", err)
	}
}

func TestCleanupOldLogs(t *testing.T) {
	dir, _ := os.MkdirTemp("", "store-test")
	defer os.RemoveAll(dir)

	s, _ := Open(filepath.Join(dir, "test.db"))
	defer s.Close()

	s.LogRequest("k", "m", "/v1", false, 200, 0, "", 0, 0)
	s.CleanupOldLogs(-1) // no-op
	s.CleanupOldLogs(0)  // no-op
	s.CleanupOldLogs(30) // should not crash
}

func TestListAccounts(t *testing.T) {
	dir, _ := os.MkdirTemp("", "store-test")
	defer os.RemoveAll(dir)

	s, _ := Open(filepath.Join(dir, "test.db"))
	defer s.Close()

	accounts, err := s.ListAccounts()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if accounts != nil {
		t.Logf("accounts: %d entries", len(accounts))
	}
}

func TestDefaultDBPath(t *testing.T) {
	path, err := DefaultDBPath()
	if err != nil {
		t.Fatalf("default db path: %v", err)
	}
	if path == "" {
		t.Errorf("expected non-empty path")
	}
}
