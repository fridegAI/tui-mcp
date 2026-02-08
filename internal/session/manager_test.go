package session

import (
	"testing"
	"time"

	"github.com/fridegai/tui-mcp/internal/interminai"
)

func TestManager_CreateGetDeleteSession(t *testing.T) {
	manager, err := NewManager(5 * time.Minute)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Stop()

	// Create session
	result := &interminai.StartResult{
		SessionID:  "/tmp/test-socket",
		SocketPath: "/tmp/test-socket",
		PID:        12345,
		AutoSocket: true,
	}

	session := manager.CreateSession(result)
	if session.ID != result.SessionID {
		t.Errorf("CreateSession ID = %q, want %q", session.ID, result.SessionID)
	}
	if session.PID != result.PID {
		t.Errorf("CreateSession PID = %d, want %d", session.PID, result.PID)
	}

	// Get session
	retrieved, ok := manager.GetSession(session.ID)
	if !ok {
		t.Fatal("GetSession returned false for existing session")
	}
	if retrieved.SocketPath != session.SocketPath {
		t.Errorf("GetSession SocketPath = %q, want %q", retrieved.SocketPath, session.SocketPath)
	}

	// Delete session
	manager.DeleteSession(session.ID)
	_, ok = manager.GetSession(session.ID)
	if ok {
		t.Error("GetSession returned true for deleted session")
	}
}

func TestManager_ListSessions(t *testing.T) {
	manager, err := NewManager(5 * time.Minute)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Stop()

	// Initially empty
	sessions := manager.ListSessions()
	if len(sessions) != 0 {
		t.Errorf("ListSessions = %d, want 0", len(sessions))
	}

	// Add sessions
	manager.CreateSession(&interminai.StartResult{SessionID: "session1", SocketPath: "session1"})
	manager.CreateSession(&interminai.StartResult{SessionID: "session2", SocketPath: "session2"})

	sessions = manager.ListSessions()
	if len(sessions) != 2 {
		t.Errorf("ListSessions = %d, want 2", len(sessions))
	}
}

func TestManager_TouchSession(t *testing.T) {
	manager, err := NewManager(5 * time.Minute)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Stop()

	result := &interminai.StartResult{
		SessionID:  "/tmp/touch-test",
		SocketPath: "/tmp/touch-test",
	}
	session := manager.CreateSession(result)
	originalTime := session.LastActive

	// Wait a bit and touch
	time.Sleep(10 * time.Millisecond)
	manager.TouchSession(session.ID)

	retrieved, _ := manager.GetSession(session.ID)
	if !retrieved.LastActive.After(originalTime) {
		t.Error("TouchSession should update LastActive time")
	}
}

func TestManager_GetSession_NotFound(t *testing.T) {
	manager, err := NewManager(5 * time.Minute)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Stop()

	_, ok := manager.GetSession("nonexistent")
	if ok {
		t.Error("GetSession should return false for nonexistent session")
	}
}
