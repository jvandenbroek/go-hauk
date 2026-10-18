package api

import (
	"sync"
	"time"
)

// viewers seen within this window count as watching; longer than any sane fetch interval
const viewerWindow = 30 * time.Second

const maxViewerIDLen = 32

// viewerTracker counts distinct viewer ids per share, in memory
type viewerTracker struct {
	mu   sync.Mutex
	seen map[string]map[string]time.Time
}

func newViewerTracker() *viewerTracker {
	return &viewerTracker{seen: make(map[string]map[string]time.Time)}
}

// touch records a viewer for a share and returns the current viewer count
func (t *viewerTracker) touch(shareID, viewerID string) int {
	now := time.Now()
	t.mu.Lock()
	defer t.mu.Unlock()

	m := t.seen[shareID]
	if m == nil {
		m = make(map[string]time.Time)
		t.seen[shareID] = m
	}
	if viewerID != "" && len(viewerID) <= maxViewerIDLen {
		m[viewerID] = now
	}
	for v, ts := range m {
		if now.Sub(ts) > viewerWindow {
			delete(m, v)
		}
	}
	if len(m) == 0 {
		delete(t.seen, shareID)
	}
	return len(m)
}
