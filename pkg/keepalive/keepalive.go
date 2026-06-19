package keepalive

import (
	"log"
	"time"
)

// CredentialRefresher defines the interface for credential refresh.
type CredentialRefresher interface {
	RefreshCredentials() error
}

// Keeper periodically checks and refreshes credentials.
type Keeper struct {
	refresher CredentialRefresher
	interval  time.Duration
	stopCh    chan struct{}
}

// NewKeeper creates a new credential keepalive.
func NewKeeper(refresher CredentialRefresher, interval time.Duration) *Keeper {
	return &Keeper{
		refresher: refresher,
		interval:  interval,
		stopCh:    make(chan struct{}),
	}
}

// Start begins the periodic credential check loop.
func (k *Keeper) Start() {
	go func() {
		ticker := time.NewTicker(k.interval)
		defer ticker.Stop()
		log.Printf("keepalive: started (interval=%v)", k.interval)
		for {
			select {
			case <-ticker.C:
				if err := k.refresher.RefreshCredentials(); err != nil {
					log.Printf("keepalive: refresh failed: %v", err)
				}
			case <-k.stopCh:
				return
			}
		}
	}()
}

// Stop signals the keepalive loop to exit.
func (k *Keeper) Stop() {
	close(k.stopCh)
}