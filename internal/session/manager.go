package session

import (
	"sync"
	"time"

	"github.com/fridegai/tui-mcp/internal/interminai"
)

// Session represents an active TUI session.
type Session struct {
	ID         string
	SocketPath string
	PID        int
	CreatedAt  time.Time
	LastActive time.Time
}

// Manager manages TUI sessions.
type Manager struct {
	sessions map[string]*Session
	client   *interminai.Client
	mu       sync.RWMutex
	timeout  time.Duration
	stopCh   chan struct{}
}

// NewManager creates a new session manager.
func NewManager(timeout time.Duration) *Manager {
	m := &Manager{
		sessions: make(map[string]*Session),
		client:   interminai.NewClient(),
		timeout:  timeout,
		stopCh:   make(chan struct{}),
	}

	// Start cleanup goroutine
	go m.cleanupLoop()

	return m
}

// Client returns the interminai client.
func (m *Manager) Client() *interminai.Client {
	return m.client
}

// CreateSession creates and stores a new session.
func (m *Manager) CreateSession(result *interminai.StartResult) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	session := &Session{
		ID:         result.SessionID,
		SocketPath: result.SocketPath,
		PID:        result.PID,
		CreatedAt:  time.Now(),
		LastActive: time.Now(),
	}

	m.sessions[session.ID] = session
	return session
}

// GetSession retrieves a session by ID.
func (m *Manager) GetSession(id string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, ok := m.sessions[id]
	if ok {
		session.LastActive = time.Now()
	}
	return session, ok
}

// DeleteSession removes a session.
func (m *Manager) DeleteSession(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.sessions, id)
}

// TouchSession updates the last active time.
func (m *Manager) TouchSession(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, ok := m.sessions[id]; ok {
		session.LastActive = time.Now()
	}
}

// cleanupLoop runs periodically to clean up inactive sessions.
func (m *Manager) cleanupLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.cleanupInactiveSessions()
		case <-m.stopCh:
			return
		}
	}
}

// cleanupInactiveSessions stops and removes sessions that have been inactive.
func (m *Manager) cleanupInactiveSessions() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for id, session := range m.sessions {
		if now.Sub(session.LastActive) > m.timeout {
			// Stop the session
			_ = m.client.Stop(session.SocketPath)
			delete(m.sessions, id)
		}
	}
}

// Stop stops the manager and cleans up all sessions.
func (m *Manager) Stop() {
	close(m.stopCh)

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, session := range m.sessions {
		_ = m.client.Stop(session.SocketPath)
	}
	m.sessions = make(map[string]*Session)
}

// ListSessions returns all active session IDs.
func (m *Manager) ListSessions() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.sessions))
	for id := range m.sessions {
		ids = append(ids, id)
	}
	return ids
}
