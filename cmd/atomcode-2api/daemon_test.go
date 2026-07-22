package main

import (
	"os"
	"testing"
)

func TestCheckRunningDaemon(t *testing.T) {
	// Without PID file, should return false
	pid, running := checkRunningDaemon()
	if running {
		t.Errorf("expected not running, got PID %d", pid)
	}
}

func TestWriteReadPIDFile(t *testing.T) {
	tmp := t.TempDir()
	daemonPIDFile = tmp + "/daemon.pid"
	defer func() { daemonPIDFile = "" }()

	data := daemonPID{PID: 12345, Port: 13457, StartedAt: "now"}
	if err := writePIDFile(data); err != nil {
		t.Fatalf("write: %v", err)
	}

	read, err := readPIDFile()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if read.PID != 12345 || read.Port != 13457 {
		t.Errorf("unexpected data: %+v", read)
	}

	removePIDFile()
	if _, err := readPIDFile(); err == nil {
		t.Errorf("expected error after remove")
	}
}

func TestRemovePIDFile(t *testing.T) {
	tmp := t.TempDir()
	daemonPIDFile = tmp + "/daemon.pid"
	defer func() { daemonPIDFile = "" }()

	writePIDFile(daemonPID{PID: 1})
	removePIDFile()
	if _, err := os.Stat(daemonPIDFile); !os.IsNotExist(err) {
		t.Errorf("expected file to be removed")
	}
}
