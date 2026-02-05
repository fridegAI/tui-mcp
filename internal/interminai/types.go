package interminai

// StartOptions contains options for starting an interminai session.
type StartOptions struct {
	Command     string            `json:"command"`
	Cwd         string            `json:"cwd,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	Cols        int               `json:"cols,omitempty"`
	Rows        int               `json:"rows,omitempty"`
	Emulator    string            `json:"emulator,omitempty"`
	Daemon      bool              `json:"daemon"`
	PtyDumpPath string            `json:"pty_dump_path,omitempty"`
}

// StartResult contains the result of starting a session.
type StartResult struct {
	SessionID  string `json:"session_id"`
	SocketPath string `json:"socket_path"`
	PID        int    `json:"pid"`
	AutoSocket bool   `json:"auto_socket"`
}

// OutputOptions contains options for getting terminal output.
type OutputOptions struct {
	Color  bool   `json:"color"`
	Cursor string `json:"cursor,omitempty"` // none, print, inverse, both
}

// OutputResult contains the terminal output.
type OutputResult struct {
	Content string       `json:"content"`
	Rows    int          `json:"rows"`
	Cols    int          `json:"cols"`
	Cursor  *CursorPos   `json:"cursor,omitempty"`
}

// CursorPos represents cursor position (1-based).
type CursorPos struct {
	Row int `json:"row"`
	Col int `json:"col"`
}

// StatusResult contains session status.
type StatusResult struct {
	Running  bool `json:"running"`
	Activity bool `json:"activity"`
	ExitCode *int `json:"exit_code,omitempty"`
}

// WaitResult contains wait result.
type WaitResult struct {
	Activity bool `json:"activity"`
	Exited   bool `json:"exited"`
	ExitCode *int `json:"exit_code,omitempty"`
}

// DebugResult contains debug information.
type DebugResult struct {
	UnhandledEscapeSequences []string      `json:"unhandled_escape_sequences"`
	Termios                  *TermiosInfo  `json:"termios,omitempty"`
}

// TermiosInfo contains terminal settings.
type TermiosInfo struct {
	Mode         string            `json:"mode"` // raw or cooked
	Flags        []string          `json:"flags"`
	Hex          map[string]string `json:"hex,omitempty"`
	ControlChars map[string]string `json:"control_chars,omitempty"`
}
