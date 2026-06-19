package proxy

import (
	"sync"
	"time"
)

const sessionTTL = 60 * time.Second

type sessionEntry struct {
	lastSeen time.Time
}

var (
	sessionMu   sync.Mutex
	sessions    = make(map[string]map[string]*sessionEntry)
	cleanupTick = time.NewTicker(30 * time.Second)
)

func init() {
	go func() {
		for range cleanupTick.C {
			cleanExpired()
		}
	}()
}

// RecordSession records that a session was active for the given apiKey.
func RecordSession(userID, sessionID string) {
	if userID == "" || sessionID == "" {
		return
	}
	sessionMu.Lock()
	defer sessionMu.Unlock()
	m, ok := sessions[userID]
	if !ok {
		m = make(map[string]*sessionEntry)
		sessions[userID] = m
	}
	if e, ok := m[sessionID]; ok {
		e.lastSeen = time.Now()
	} else {
		m[sessionID] = &sessionEntry{lastSeen: time.Now()}
	}
}

// GetActiveSessions returns the number of distinct active sessions for the given userID.
func GetActiveSessions(userID string) int64 {
	sessionMu.Lock()
	defer sessionMu.Unlock()
	m, ok := sessions[userID]
	if !ok {
		return 0
	}
	now := time.Now()
	count := int64(0)
	for _, e := range m {
		if now.Sub(e.lastSeen) < sessionTTL {
			count++
		}
	}
	return count
}

func cleanExpired() {
	sessionMu.Lock()
	defer sessionMu.Unlock()
	now := time.Now()
	for uid, m := range sessions {
		for sid, e := range m {
			if now.Sub(e.lastSeen) >= sessionTTL {
				delete(m, sid)
			}
		}
		if len(m) == 0 {
			delete(sessions, uid)
		}
	}
}