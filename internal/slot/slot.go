// Package slot owns the per-project index lifecycle. Each Slot wraps one
// project root, its ProjectIndex, and a monotonic CacheGen counter that
// downstream caches use as an invalidation key.
package slot

import (
	"sync"
	"sync/atomic"

	"token-savior-go/internal/models"
)

// Slot is a per-project state container.
type Slot struct {
	Root     string
	Index    *models.ProjectIndex
	CacheGen atomic.Uint64
	mu       sync.RWMutex
}

// BumpCacheGen increments the cache generation. Call after any index
// mutation (full rebuild, incremental update, edit).
func (s *Slot) BumpCacheGen() { s.CacheGen.Add(1) }

// Lock acquires the write lock; callers mutating Index MUST hold this.
func (s *Slot) Lock()   { s.mu.Lock() }
func (s *Slot) Unlock() { s.mu.Unlock() }

// RLock acquires the read lock; query handlers reading Index hold this.
func (s *Slot) RLock()   { s.mu.RLock() }
func (s *Slot) RUnlock() { s.mu.RUnlock() }
