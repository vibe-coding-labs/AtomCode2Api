package logrot

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriter(t *testing.T) {
	dir, err := os.MkdirTemp("", "logrot-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	cfg := Config{Dir: dir, Name: "test", MaxSize: 100, MaxFiles: 3}
	w, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Write enough to trigger rotation
	data := make([]byte, 50)
	for i := 0; i < 10; i++ {
		_, err := w.Write(data)
		if err != nil {
			t.Fatal(err)
		}
	}
	w.Close()

	// Check that rotated files exist
	entries, _ := os.ReadDir(dir)
	var logs int
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".log" {
			logs++
		}
	}
	if logs < 2 {
		t.Errorf("expected at least 2 log files, got %d: %v", logs, entries)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig("/tmp", "test")
	if cfg.MaxSize != defaultMaxSize {
		t.Errorf("expected max size %d, got %d", defaultMaxSize, cfg.MaxSize)
	}
	if cfg.MaxFiles != defaultMaxFiles {
		t.Errorf("expected max files %d, got %d", defaultMaxFiles, cfg.MaxFiles)
	}
}

func TestTruncateFileIfNeeded(t *testing.T) {
	f, _ := os.CreateTemp("", "logrot-trunc")
	path := f.Name()
	f.WriteString("hello world")
	f.Close()
	defer os.Remove(path)

	TruncateFileIfNeeded(path, 5)
	info, _ := os.Stat(path)
	if info.Size() != 0 {
		t.Errorf("file should be truncated to 0, got %d", info.Size())
	}
}
