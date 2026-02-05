package interminai

import (
	"testing"
)

func TestParseStartOutput(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		wantPID  int
		wantPath string
		wantAuto bool
		wantErr  bool
	}{
		{
			name: "valid output",
			output: `PID: 12345
Socket: /tmp/interminai-abc123/socket
Auto-generated: true`,
			wantPID:  12345,
			wantPath: "/tmp/interminai-abc123/socket",
			wantAuto: true,
			wantErr:  false,
		},
		{
			name: "custom socket",
			output: `PID: 99999
Socket: /custom/path/socket
Auto-generated: false`,
			wantPID:  99999,
			wantPath: "/custom/path/socket",
			wantAuto: false,
			wantErr:  false,
		},
		{
			name:    "missing socket",
			output:  "PID: 123\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseStartOutput(tt.output)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.PID != tt.wantPID {
				t.Errorf("PID = %d, want %d", result.PID, tt.wantPID)
			}
			if result.SocketPath != tt.wantPath {
				t.Errorf("SocketPath = %q, want %q", result.SocketPath, tt.wantPath)
			}
			if result.AutoSocket != tt.wantAuto {
				t.Errorf("AutoSocket = %v, want %v", result.AutoSocket, tt.wantAuto)
			}
		})
	}
}

func TestParseStatusOutput(t *testing.T) {
	tests := []struct {
		name         string
		output       string
		wantRunning  bool
		wantActivity bool
		wantExitCode *int
	}{
		{
			name: "running with activity",
			output: `Running: true
Activity: true`,
			wantRunning:  true,
			wantActivity: true,
		},
		{
			name: "exited",
			output: `Running: false
Activity: false
Exit code: 0`,
			wantRunning:  false,
			wantActivity: false,
			wantExitCode: intPtr(0),
		},
		{
			name: "exited with error",
			output: `Running: false
Activity: false
Exit code: 1`,
			wantRunning:  false,
			wantActivity: false,
			wantExitCode: intPtr(1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseStatusOutput(tt.output)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Running != tt.wantRunning {
				t.Errorf("Running = %v, want %v", result.Running, tt.wantRunning)
			}
			if result.Activity != tt.wantActivity {
				t.Errorf("Activity = %v, want %v", result.Activity, tt.wantActivity)
			}
			if tt.wantExitCode != nil {
				if result.ExitCode == nil || *result.ExitCode != *tt.wantExitCode {
					t.Errorf("ExitCode = %v, want %v", result.ExitCode, *tt.wantExitCode)
				}
			}
		})
	}
}

func TestParseWaitOutput(t *testing.T) {
	tests := []struct {
		name         string
		output       string
		wantActivity bool
		wantExited   bool
	}{
		{
			name: "activity detected",
			output: `Terminal activity: true
Application exited: false`,
			wantActivity: true,
			wantExited:   false,
		},
		{
			name: "application exited",
			output: `Terminal activity: false
Application exited: true`,
			wantActivity: false,
			wantExited:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseWaitOutput(tt.output)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Activity != tt.wantActivity {
				t.Errorf("Activity = %v, want %v", result.Activity, tt.wantActivity)
			}
			if result.Exited != tt.wantExited {
				t.Errorf("Exited = %v, want %v", result.Exited, tt.wantExited)
			}
		})
	}
}

func TestParseOutputResult(t *testing.T) {
	tests := []struct {
		name       string
		output     string
		cursorMode string
		wantRow    int
		wantCol    int
	}{
		{
			name:       "no cursor",
			output:     "Hello World\n~\n~\n",
			cursorMode: "none",
			wantRow:    0,
			wantCol:    0,
		},
		{
			name:       "with cursor print",
			output:     "Cursor: row 5, col 10\nHello World\n~\n",
			cursorMode: "print",
			wantRow:    5,
			wantCol:    10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseOutputResult(tt.output, tt.cursorMode)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantRow > 0 && (result.Cursor == nil || result.Cursor.Row != tt.wantRow) {
				t.Errorf("Cursor.Row = %v, want %d", result.Cursor, tt.wantRow)
			}
			if tt.wantCol > 0 && (result.Cursor == nil || result.Cursor.Col != tt.wantCol) {
				t.Errorf("Cursor.Col = %v, want %d", result.Cursor, tt.wantCol)
			}
		})
	}
}

func TestParseDebugOutput(t *testing.T) {
	output := `Unhandled escape sequences:
  \e\ (1b5c)
  \e[?1049h
Termios:
  Mode: raw
  Flags: OPOST ECHO`

	result, err := parseDebugOutput(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.UnhandledEscapeSequences) != 2 {
		t.Errorf("UnhandledEscapeSequences len = %d, want 2", len(result.UnhandledEscapeSequences))
	}

	if result.Termios == nil {
		t.Fatal("Termios is nil")
	}
	if result.Termios.Mode != "raw" {
		t.Errorf("Termios.Mode = %q, want %q", result.Termios.Mode, "raw")
	}
}

func intPtr(i int) *int {
	return &i
}
