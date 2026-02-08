package tools

import (
	"context"
	"testing"
	"time"

	"github.com/fridegai/tui-mcp/internal/interminai"
	"github.com/fridegai/tui-mcp/internal/security"
	"github.com/fridegai/tui-mcp/internal/session"
	"github.com/mark3labs/mcp-go/mcp"
)

// TestHandleStart_BlockedCommand tests that blocked commands are rejected
func TestHandleStart_BlockedCommand(t *testing.T) {
	manager, err := session.NewManager(5 * time.Minute)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Stop()

	allowlist := security.NewAllowlist(true)
	handler := NewHandler(manager, allowlist)

	// Create a request with a blocked command
	req := mcp.CallToolRequest{}
	req.Params.Name = "tui.start"
	req.Params.Arguments = map[string]interface{}{
		"command": "rm -rf /",
	}

	result, err := handler.handleStart(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return an error result
	if !result.IsError {
		t.Error("expected error result for blocked command")
	}
}

// TestHandleStart_AllowedCommand tests that allowed commands pass validation
func TestHandleStart_AllowedCommand_Validation(t *testing.T) {
	manager, err := session.NewManager(5 * time.Minute)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Stop()

	allowlist := security.NewAllowlist(true)
	handler := NewHandler(manager, allowlist)

	// Create a request with an allowed command
	req := mcp.CallToolRequest{}
	req.Params.Name = "tui.start"
	req.Params.Arguments = map[string]interface{}{
		"command": "ls -la",
	}

	// This will fail because interminai isn't running, but it should pass validation
	result, err := handler.handleStart(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should NOT be a "blocked" error - it should be an interminai execution error
	if result.IsError {
		// Check it's not a blocked command error
		for _, content := range result.Content {
			if textContent, ok := content.(mcp.TextContent); ok {
				if textContent.Text == "command contains blocked pattern: rm -rf" {
					t.Error("ls -la should not be blocked")
				}
			}
		}
	}
}

// TestHandleInput_SessionNotFound tests error handling for missing session
func TestHandleInput_SessionNotFound(t *testing.T) {
	manager, err := session.NewManager(5 * time.Minute)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Stop()

	handler := NewHandler(manager, nil)

	req := mcp.CallToolRequest{}
	req.Params.Name = "tui.input"
	req.Params.Arguments = map[string]interface{}{
		"session_id": "nonexistent-session",
		"text":       "hello",
	}

	result, err := handler.handleInput(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error result for nonexistent session")
	}
}

// TestHandleOutput_SessionNotFound tests error handling for missing session
func TestHandleOutput_SessionNotFound(t *testing.T) {
	manager, err := session.NewManager(5 * time.Minute)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Stop()

	handler := NewHandler(manager, nil)

	req := mcp.CallToolRequest{}
	req.Params.Name = "tui.output"
	req.Params.Arguments = map[string]interface{}{
		"session_id": "nonexistent-session",
	}

	result, err := handler.handleOutput(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error result for nonexistent session")
	}
}

// TestHandleStatus_SessionNotFound tests error handling for missing session
func TestHandleStatus_SessionNotFound(t *testing.T) {
	manager, err := session.NewManager(5 * time.Minute)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Stop()

	handler := NewHandler(manager, nil)

	req := mcp.CallToolRequest{}
	req.Params.Name = "tui.status"
	req.Params.Arguments = map[string]interface{}{
		"session_id": "nonexistent-session",
	}

	result, err := handler.handleStatus(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error result for nonexistent session")
	}
}

// TestHandleStop_SessionNotFound tests error handling for missing session
func TestHandleStop_SessionNotFound(t *testing.T) {
	manager, err := session.NewManager(5 * time.Minute)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Stop()

	handler := NewHandler(manager, nil)

	req := mcp.CallToolRequest{}
	req.Params.Name = "tui.stop"
	req.Params.Arguments = map[string]interface{}{
		"session_id": "nonexistent-session",
	}

	result, err := handler.handleStop(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error result for nonexistent session")
	}
}

// TestHandleInput_PasswordMode tests password delegation message
func TestHandleInput_PasswordMode(t *testing.T) {
	manager, err := session.NewManager(5 * time.Minute)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Stop()

	handler := NewHandler(manager, nil)

	// Create a fake session first
	manager.CreateSession(&interminai.StartResult{
		SessionID:  "test-session",
		SocketPath: "/tmp/test-socket",
		PID:        12345,
	})

	req := mcp.CallToolRequest{}
	req.Params.Name = "tui.input"
	req.Params.Arguments = map[string]interface{}{
		"session_id": "test-session",
		"text":       "password123",
		"password":   true,
	}

	result, err := handler.handleInput(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return a message about password input
	if result.IsError {
		t.Error("password mode should not return error")
	}
}

// TestHandleResize_MissingParams tests missing required parameters
func TestHandleResize_MissingParams(t *testing.T) {
	manager, err := session.NewManager(5 * time.Minute)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Stop()

	handler := NewHandler(manager, nil)

	// Missing cols and rows
	req := mcp.CallToolRequest{}
	req.Params.Name = "tui.resize_screen"
	req.Params.Arguments = map[string]interface{}{
		"session_id": "test-session",
	}

	result, err := handler.handleResize(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error result for missing required params")
	}
}

// TestHandleSignal_MissingParams tests missing required parameters
func TestHandleSignal_MissingSignal(t *testing.T) {
	manager, err := session.NewManager(5 * time.Minute)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Stop()

	handler := NewHandler(manager, nil)

	req := mcp.CallToolRequest{}
	req.Params.Name = "tui.signal"
	req.Params.Arguments = map[string]interface{}{
		"session_id": "test-session",
	}

	result, err := handler.handleSignal(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error result for missing signal param")
	}
}

// TestHandleObserve_SessionNotFound tests error handling for missing session
func TestHandleObserve_SessionNotFound(t *testing.T) {
	manager, err := session.NewManager(5 * time.Minute)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Stop()

	handler := NewHandler(manager, nil)

	req := mcp.CallToolRequest{}
	req.Params.Name = "tui.observe"
	req.Params.Arguments = map[string]interface{}{
		"session_id": "nonexistent-session",
	}

	result, err := handler.handleObserve(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error result for nonexistent session")
	}
}

// TestHandleDebug_SessionNotFound tests error handling for missing session
func TestHandleDebug_SessionNotFound(t *testing.T) {
	manager, err := session.NewManager(5 * time.Minute)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Stop()

	handler := NewHandler(manager, nil)

	req := mcp.CallToolRequest{}
	req.Params.Name = "tui.debug"
	req.Params.Arguments = map[string]interface{}{
		"session_id": "nonexistent-session",
	}

	result, err := handler.handleDebug(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error result for nonexistent session")
	}
}

// TestHandleWait_SessionNotFound tests error handling for missing session
func TestHandleWait_SessionNotFound(t *testing.T) {
	manager, err := session.NewManager(5 * time.Minute)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Stop()

	handler := NewHandler(manager, nil)

	req := mcp.CallToolRequest{}
	req.Params.Name = "tui.wait"
	req.Params.Arguments = map[string]interface{}{
		"session_id": "nonexistent-session",
	}

	result, err := handler.handleWait(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error result for nonexistent session")
	}
}
