package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/fridegai/tui-mcp/internal/interminai"
	"github.com/fridegai/tui-mcp/internal/security"
	"github.com/fridegai/tui-mcp/internal/session"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Handler holds dependencies for tool handlers.
type Handler struct {
	manager   *session.Manager
	allowlist *security.Allowlist
}

// NewHandler creates a new tool handler.
func NewHandler(manager *session.Manager, allowlist *security.Allowlist) *Handler {
	return &Handler{
		manager:   manager,
		allowlist: allowlist,
	}
}

// RegisterTools registers all TUI tools with the MCP server.
func (h *Handler) RegisterTools(s *server.MCPServer) {
	// tui.start
	s.AddTool(mcp.NewTool("tui.start",
		mcp.WithDescription("Start a TUI session. Returns session_id to use with other tools."),
		mcp.WithString("command", mcp.Required(), mcp.Description("Command to run")),
		mcp.WithString("cwd", mcp.Description("Working directory")),
		mcp.WithObject("env", mcp.Description("Environment variables")),
		mcp.WithNumber("cols", mcp.Description("Terminal columns (default: 80)")),
		mcp.WithNumber("rows", mcp.Description("Terminal rows (default: 24)")),
		mcp.WithString("emulator", mcp.Description("Terminal emulator (default: xterm)")),
	), h.handleStart)

	// tui.input
	s.AddTool(mcp.NewTool("tui.input",
		mcp.WithDescription("Send input to a TUI session. Use \\r for Enter, \\e for Escape; for arrow keys, e.g. up arrow, use \\e[A in Normal Cursor Mode, use \\eOA in Application Cursor Mode."),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("Session ID from tui.start")),
		mcp.WithString("text", mcp.Required(), mcp.Description("Text to send (supports C-style escapes and ANSI Escape Sequences: \\r \\n \\e \\t \\xHH)")),
		mcp.WithBoolean("password", mcp.Description("If true, prompt user to enter password manually")),
	), h.handleInput)

	// tui.output
	s.AddTool(mcp.NewTool("tui.output",
		mcp.WithDescription("Get the current terminal screen content."),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("Session ID")),
		mcp.WithBoolean("color", mcp.Description("Include ANSI color codes (default: false)")),
		mcp.WithString("cursor", mcp.Description("Cursor mode: none, print, inverse, both (default: print)")),
	), h.handleOutput)

	// tui.status
	s.AddTool(mcp.NewTool("tui.status",
		mcp.WithDescription("Check if session is running and has new activity."),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("Session ID")),
		mcp.WithBoolean("quiet", mcp.Description("Only return running state")),
	), h.handleStatus)

	// tui.wait
	s.AddTool(mcp.NewTool("tui.wait",
		mcp.WithDescription("Wait for terminal activity or process exit. Use it for long running commands to avoid polling."),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("Session ID")),
		mcp.WithString("mode", mcp.Description("Wait mode: activity or exit")),
		mcp.WithNumber("timeout_ms", mcp.Description("Timeout in milliseconds")),
	), h.handleWait)

	// tui.resize
	s.AddTool(mcp.NewTool("tui.resize",
		mcp.WithDescription("Resize the terminal window. Sends SIGWINCH to trigger redraw."),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("Session ID")),
		mcp.WithNumber("cols", mcp.Required(), mcp.Description("New column count")),
		mcp.WithNumber("rows", mcp.Required(), mcp.Description("New row count")),
	), h.handleResize)

	// tui.signal
	s.AddTool(mcp.NewTool("tui.signal",
		mcp.WithDescription("Send a signal to the process. For example, use SIGINT for Ctrl+C."),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("Session ID")),
		mcp.WithString("signal", mcp.Required(), mcp.Description("Signal: SIGINT, SIGTERM, SIGKILL, SIGQUIT, or number")),
	), h.handleSignal)

	// tui.stop
	s.AddTool(mcp.NewTool("tui.stop",
		mcp.WithDescription("Stop and clean up a session. Always call when a session is no longer needed."),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("Session ID")),
	), h.handleStop)

	// tui.debug
	s.AddTool(mcp.NewTool("tui.debug",
		mcp.WithDescription("Get debug info: termios mode, flags, unhandled escapes. Use it to determine Normal Cursor Mode vs Application Cursor Mode."),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("Session ID")),
		mcp.WithBoolean("clear", mcp.Description("Clear unhandled escape buffer")),
	), h.handleDebug)

	// tui.observe
	s.AddTool(mcp.NewTool("tui.observe",
		mcp.WithDescription("Convenience: get screen output and status in one call."),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("Session ID")),
	), h.handleObserve)
}

// handleStart handles tui.start
func (h *Handler) handleStart(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	command, err := req.RequireString("command")
	if err != nil {
		return mcp.NewToolResultError("command is required"), nil
	}

	// Validate command
	if h.allowlist != nil {
		if ok, reason := h.allowlist.Validate(command); !ok {
			return mcp.NewToolResultError(reason), nil
		}
	}

	opts := interminai.StartOptions{
		Command: command,
		Daemon:  true,
		Cols:    req.GetInt("cols", 80),
		Rows:    req.GetInt("rows", 24),
	}

	opts.Cwd = req.GetString("cwd", "")
	opts.Emulator = req.GetString("emulator", "xterm")

	result, err := h.manager.Client().Start(ctx, opts)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Store session
	h.manager.CreateSession(result)

	return mcp.NewToolResultText(fmt.Sprintf(
		`{"session_id": "%s", "socket_path": "%s", "pid": %d, "auto_socket": %t}`,
		result.SessionID, result.SocketPath, result.PID, result.AutoSocket,
	)), nil
}

// handleInput handles tui.input
func (h *Handler) handleInput(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID, err := req.RequireString("session_id")
	if err != nil {
		return mcp.NewToolResultError("session_id is required"), nil
	}

	text, err := req.RequireString("text")
	if err != nil {
		return mcp.NewToolResultError("text is required"), nil
	}

	password := req.GetBool("password", false)

	sess, ok := h.manager.GetSession(sessionID)
	if !ok {
		return mcp.NewToolResultError("session not found: " + sessionID), nil
	}

	if password {
		return mcp.NewToolResultText(
			`{"message": "Password input required. User must run: interminai input --socket ` + sess.SocketPath + ` --password"}`,
		), nil
	}

	err = h.manager.Client().Input(ctx, sess.SocketPath, text, false)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	h.manager.TouchSession(sessionID)
	return mcp.NewToolResultText(`{"success": true}`), nil
}

// handleOutput handles tui.output
func (h *Handler) handleOutput(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID, err := req.RequireString("session_id")
	if err != nil {
		return mcp.NewToolResultError("session_id is required"), nil
	}

	sess, ok := h.manager.GetSession(sessionID)
	if !ok {
		return mcp.NewToolResultError("session not found: " + sessionID), nil
	}

	opts := interminai.OutputOptions{
		Color:  req.GetBool("color", false),
		Cursor: req.GetString("cursor", "print"),
	}

	result, err := h.manager.Client().Output(ctx, sess.SocketPath, opts)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	h.manager.TouchSession(sessionID)

	// Build response
	cursorJSON := "null"
	if result.Cursor != nil {
		cursorJSON = fmt.Sprintf(`{"row": %d, "col": %d}`, result.Cursor.Row, result.Cursor.Col)
	}

	return mcp.NewToolResultText(fmt.Sprintf(
		`{"content": %q, "rows": %d, "cols": %d, "cursor": %s}`,
		result.Content, result.Rows, result.Cols, cursorJSON,
	)), nil
}

// handleStatus handles tui.status
func (h *Handler) handleStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID, err := req.RequireString("session_id")
	if err != nil {
		return mcp.NewToolResultError("session_id is required"), nil
	}

	quiet := req.GetBool("quiet", false)

	sess, ok := h.manager.GetSession(sessionID)
	if !ok {
		return mcp.NewToolResultError("session not found: " + sessionID), nil
	}

	result, err := h.manager.Client().Status(ctx, sess.SocketPath, quiet)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	exitCodeJSON := "null"
	if result.ExitCode != nil {
		exitCodeJSON = fmt.Sprintf("%d", *result.ExitCode)
	}

	return mcp.NewToolResultText(fmt.Sprintf(
		`{"running": %t, "activity": %t, "exit_code": %s}`,
		result.Running, result.Activity, exitCodeJSON,
	)), nil
}

// handleWait handles tui.wait
func (h *Handler) handleWait(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID, err := req.RequireString("session_id")
	if err != nil {
		return mcp.NewToolResultError("session_id is required"), nil
	}

	mode := req.GetString("mode", "activity")
	quiet := mode == "exit"
	timeoutMs := req.GetInt("timeout_ms", 0)

	sess, ok := h.manager.GetSession(sessionID)
	if !ok {
		return mcp.NewToolResultError("session not found: " + sessionID), nil
	}

	// Create timeout context if specified
	var waitCtx context.Context = ctx
	if timeoutMs > 0 {
		var cancel context.CancelFunc
		waitCtx, cancel = context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
		defer cancel()
	}

	result, err := h.manager.Client().Wait(waitCtx, sess.SocketPath, quiet, timeoutMs)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	exitCodeJSON := "null"
	if result.ExitCode != nil {
		exitCodeJSON = fmt.Sprintf("%d", *result.ExitCode)
	}

	return mcp.NewToolResultText(fmt.Sprintf(
		`{"activity": %t, "exited": %t, "exit_code": %s}`,
		result.Activity, result.Exited, exitCodeJSON,
	)), nil
}

// handleResize handles tui.resize_screen
func (h *Handler) handleResize(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID, err := req.RequireString("session_id")
	if err != nil {
		return mcp.NewToolResultError("session_id is required"), nil
	}

	cols, err := req.RequireInt("cols")
	if err != nil {
		return mcp.NewToolResultError("cols is required"), nil
	}

	rows, err := req.RequireInt("rows")
	if err != nil {
		return mcp.NewToolResultError("rows is required"), nil
	}

	sess, ok := h.manager.GetSession(sessionID)
	if !ok {
		return mcp.NewToolResultError("session not found: " + sessionID), nil
	}

	err = h.manager.Client().Resize(ctx, sess.SocketPath, cols, rows)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(`{"success": true}`), nil
}

// handleSignal handles tui.signal
func (h *Handler) handleSignal(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID, err := req.RequireString("session_id")
	if err != nil {
		return mcp.NewToolResultError("session_id is required"), nil
	}

	signal, err := req.RequireString("signal")
	if err != nil {
		return mcp.NewToolResultError("signal is required"), nil
	}

	sess, ok := h.manager.GetSession(sessionID)
	if !ok {
		return mcp.NewToolResultError("session not found: " + sessionID), nil
	}

	err = h.manager.Client().Kill(ctx, sess.SocketPath, signal)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(`{"success": true}`), nil
}

// handleStop handles tui.stop
func (h *Handler) handleStop(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID, err := req.RequireString("session_id")
	if err != nil {
		return mcp.NewToolResultError("session_id is required"), nil
	}

	sess, ok := h.manager.GetSession(sessionID)
	if !ok {
		return mcp.NewToolResultError("session not found: " + sessionID), nil
	}

	err = h.manager.Client().Stop(sess.SocketPath)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	h.manager.DeleteSession(sessionID)

	return mcp.NewToolResultText(`{"success": true}`), nil
}

// handleDebug handles tui.debug
func (h *Handler) handleDebug(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID, err := req.RequireString("session_id")
	if err != nil {
		return mcp.NewToolResultError("session_id is required"), nil
	}

	clear := req.GetBool("clear", false)

	sess, ok := h.manager.GetSession(sessionID)
	if !ok {
		return mcp.NewToolResultError("session not found: " + sessionID), nil
	}

	result, err := h.manager.Client().Debug(ctx, sess.SocketPath, clear)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Build response
	escapesJSON := "[]"
	if len(result.UnhandledEscapeSequences) > 0 {
		escapesJSON = `["` + result.UnhandledEscapeSequences[0]
		for _, e := range result.UnhandledEscapeSequences[1:] {
			escapesJSON += `", "` + e
		}
		escapesJSON += `"]`
	}

	termiosJSON := "null"
	if result.Termios != nil {
		flagsJSON := "[]"
		if len(result.Termios.Flags) > 0 {
			flagsJSON = `["` + result.Termios.Flags[0]
			for _, f := range result.Termios.Flags[1:] {
				flagsJSON += `", "` + f
			}
			flagsJSON += `"]`
		}
		termiosJSON = fmt.Sprintf(`{"mode": "%s", "flags": %s}`, result.Termios.Mode, flagsJSON)
	}

	return mcp.NewToolResultText(fmt.Sprintf(
		`{"unhandled_escape_sequences": %s, "termios": %s}`,
		escapesJSON, termiosJSON,
	)), nil
}

// handleObserve handles tui.observe (convenience wrapper)
func (h *Handler) handleObserve(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID, err := req.RequireString("session_id")
	if err != nil {
		return mcp.NewToolResultError("session_id is required"), nil
	}

	sess, ok := h.manager.GetSession(sessionID)
	if !ok {
		return mcp.NewToolResultError("session not found: " + sessionID), nil
	}

	// Get output
	output, err := h.manager.Client().Output(ctx, sess.SocketPath, interminai.OutputOptions{Color: false})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Get status
	status, err := h.manager.Client().Status(ctx, sess.SocketPath, false)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	h.manager.TouchSession(sessionID)

	return mcp.NewToolResultText(fmt.Sprintf(
		`{"screen": %q, "running": %t, "activity": %t}`,
		output.Content, status.Running, status.Activity,
	)), nil
}
