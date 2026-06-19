package logrot

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	defaultMaxSize  = 10 << 20 // 10 MB
	defaultMaxFiles = 5
)

// Config for log rotation.
type Config struct {
	Dir        string
	Name       string
	MaxSize    int64
	MaxFiles   int
}

// DefaultConfig returns a default Config.
func DefaultConfig(dir, name string) Config {
	return Config{Dir: dir, Name: name, MaxSize: defaultMaxSize, MaxFiles: defaultMaxFiles}
}

// Writer is an io.WriteCloser that rotates logs automatically.
type Writer struct {
	mu     sync.Mutex
	cfg    Config
	file   *os.File
	size   int64
}

// New creates a new rotating log writer.
func New(cfg Config) (*Writer, error) {
	if err := os.MkdirAll(cfg.Dir, 0755); err != nil {
		return nil, fmt.Errorf("logrot: create dir %s: %w", cfg.Dir, err)
	}
	w := &Writer{cfg: cfg}
	if err := w.open(); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *Writer) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		if err := w.open(); err != nil {
			return 0, err
		}
	}
	n, err := w.file.Write(p)
	if err == nil {
		w.size += int64(n)
	}
	if w.size >= w.cfg.MaxSize {
		w.rotate()
	}
	return n, err
}

func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

func (w *Writer) open() error {
	path := filepath.Join(w.cfg.Dir, w.cfg.Name+".log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("logrot: open %s: %w", path, err)
	}
	info, err := f.Stat()
	if err == nil {
		w.size = info.Size()
	} else {
		w.size = 0
	}
	w.file = f
	return nil
}

func (w *Writer) rotate() {
	if w.file != nil {
		w.file.Close()
		w.file = nil
	}
	w.size = 0

	base := filepath.Join(w.cfg.Dir, w.cfg.Name)
	ts := time.Now().Format("20060102-150405")

	// Rename current log
	os.Rename(base+".log", base+"."+ts+".log")

	// Cleanup old files
	entries, _ := os.ReadDir(w.cfg.Dir)
	var logs []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), w.cfg.Name+".") && strings.HasSuffix(e.Name(), ".log") && e.Name() != w.cfg.Name+".log" {
			logs = append(logs, e.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(logs)))
	for i := w.cfg.MaxFiles; i < len(logs); i++ {
		os.Remove(filepath.Join(w.cfg.Dir, logs[i]))
	}

	// Reopen
	w.open()
}

// TruncateFileIfNeeded truncates a file if it's larger than maxSize.
func TruncateFileIfNeeded(path string, maxSize int64) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	if info.Size() > maxSize {
		os.Truncate(path, 0)
	}
}

var _ io.WriteCloser = (*Writer)(nil)
