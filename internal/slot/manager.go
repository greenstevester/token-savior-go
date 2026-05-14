package slot

import (
	"fmt"
	"strings"
	"sync"

	"token-savior-go/internal/indexer"
)

// Manager owns the dict of project Slots and tracks the currently-active root.
//
// Single-threaded creation expected at server start, but Get/Active/Switch
// are safe for concurrent calls from MCP tool handlers.
type Manager struct {
	mu     sync.RWMutex
	slots  map[string]*Slot
	active string
}

// NewManager returns an empty manager.
func NewManager() *Manager {
	return &Manager{slots: make(map[string]*Slot)}
}

// RegisterRoot indexes a project root and stores it in the manager. The
// first root registered also becomes active. Re-registering an existing
// root is a no-op (returns nil).
func (m *Manager) RegisterRoot(root string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.slots[root]; exists {
		return nil
	}

	idx, err := indexer.NewProjectIndexer(root).Build()
	if err != nil {
		return fmt.Errorf("index %s: %w", root, err)
	}
	s := &Slot{Root: root, Index: idx}
	s.BumpCacheGen()
	m.slots[root] = s
	if m.active == "" {
		m.active = root
	}
	return nil
}

// Get returns the slot for root, or (nil, false) if not registered.
func (m *Manager) Get(root string) (*Slot, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.slots[root]
	return s, ok
}

// Active returns the currently-active slot. Returns nil if no roots are
// registered yet — callers MUST handle this (most won't have to since
// main.go registers at least one root before serving requests).
func (m *Manager) Active() *Slot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.active == "" {
		return nil
	}
	return m.slots[m.active]
}

// Switch changes the active slot to root. Idempotent — switching to the
// current active root returns the existing slot without changes.
func (m *Manager) Switch(root string) (*Slot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.slots[root]
	if !ok {
		return nil, fmt.Errorf("unknown workspace root: %s", root)
	}
	m.active = root
	return s, nil
}

// Roots returns the registered roots in registration order. (Map iteration
// order is randomised in Go, but the slice is built deterministically by
// caller convention: tests sort before comparing.)
func (m *Manager) Roots() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]string, 0, len(m.slots))
	for r := range m.slots {
		out = append(out, r)
	}
	return out
}

// ParseWorkspaceRoots splits the WORKSPACE_ROOTS env-var value on commas
// and trims whitespace. Empty input returns an empty slice.
func ParseWorkspaceRoots(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
