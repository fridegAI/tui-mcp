package interminai

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const BinaryPath = "/usr/local/bin/interminai"

// Client wraps the interminai CLI.
type Client struct {
	binaryPath string
}

// NewClient creates a new interminai client.
func NewClient() *Client {
	return &Client{binaryPath: BinaryPath}
}

// Start starts an interminai session.
func (c *Client) Start(opts StartOptions) (*StartResult, error) {
	args := []string{"start"}

	// Size
	cols := opts.Cols
	if cols == 0 {
		cols = 80
	}
	rows := opts.Rows
	if rows == 0 {
		rows = 24
	}
	args = append(args, "--size", fmt.Sprintf("%dx%d", cols, rows))

	// Emulator
	if opts.Emulator != "" {
		args = append(args, "--emulator", opts.Emulator)
	}

	// PTY dump
	if opts.PtyDumpPath != "" {
		args = append(args, "--pty-dump", opts.PtyDumpPath)
	}

	// Command (after --)
	args = append(args, "--")
	args = append(args, "sh", "-c", opts.Command)

	cmd := exec.Command(c.binaryPath, args...)
	if opts.Cwd != "" {
		cmd.Dir = opts.Cwd
	}

	// Set environment
	if len(opts.Env) > 0 {
		cmd.Env = cmd.Environ()
		for k, v := range opts.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("interminai start failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("interminai start failed: %w", err)
	}

	return parseStartOutput(string(output))
}

// parseStartOutput parses the output from interminai start.
// Example output:
//   PID: 12345
//   Socket: /tmp/interminai-xxx/socket
//   Auto-generated: true
func parseStartOutput(output string) (*StartResult, error) {
	result := &StartResult{}
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "PID:") {
			pidStr := strings.TrimSpace(strings.TrimPrefix(line, "PID:"))
			pid, err := strconv.Atoi(pidStr)
			if err != nil {
				return nil, fmt.Errorf("invalid PID: %s", pidStr)
			}
			result.PID = pid
		} else if strings.HasPrefix(line, "Socket:") {
			result.SocketPath = strings.TrimSpace(strings.TrimPrefix(line, "Socket:"))
			// Use socket path as session ID
			result.SessionID = result.SocketPath
		} else if strings.HasPrefix(line, "Auto-generated:") {
			result.AutoSocket = strings.TrimSpace(strings.TrimPrefix(line, "Auto-generated:")) == "true"
		}
	}

	if result.SocketPath == "" {
		return nil, fmt.Errorf("failed to parse socket path from output: %s", output)
	}

	return result, nil
}

// Input sends input to a session.
func (c *Client) Input(socketPath, text string, password bool) error {
	args := []string{"input", "--socket", socketPath}
	if password {
		args = append(args, "--password")
	} else {
		args = append(args, "--text", text)
	}

	cmd := exec.Command(c.binaryPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("interminai input failed: %s", string(output))
	}
	return nil
}

// Output gets the current terminal output.
func (c *Client) Output(socketPath string, opts OutputOptions) (*OutputResult, error) {
	args := []string{"output", "--socket", socketPath}

	if !opts.Color {
		args = append(args, "--no-color")
	}
	if opts.Cursor != "" && opts.Cursor != "none" {
		args = append(args, "--cursor", opts.Cursor)
	}

	cmd := exec.Command(c.binaryPath, args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("interminai output failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("interminai output failed: %w", err)
	}

	return parseOutputResult(string(output), opts.Cursor)
}

// parseOutputResult parses output command result.
func parseOutputResult(output string, cursorMode string) (*OutputResult, error) {
	result := &OutputResult{
		Content: output,
		Cols:    80,
		Rows:    24,
	}

	// Parse cursor position if cursor mode is "print" or "both"
	if cursorMode == "print" || cursorMode == "both" {
		// Look for "Cursor: row X, col Y" at the start
		re := regexp.MustCompile(`^Cursor: row (\d+), col (\d+)\n`)
		matches := re.FindStringSubmatch(output)
		if len(matches) == 3 {
			row, _ := strconv.Atoi(matches[1])
			col, _ := strconv.Atoi(matches[2])
			result.Cursor = &CursorPos{Row: row, Col: col}
			result.Content = re.ReplaceAllString(output, "")
		}
	}

	// Count rows from content
	lines := strings.Split(result.Content, "\n")
	result.Rows = len(lines)

	return result, nil
}

// Status gets the session status.
func (c *Client) Status(socketPath string, quiet bool) (*StatusResult, error) {
	args := []string{"status", "--socket", socketPath}
	if quiet {
		args = append(args, "--quiet")
	}

	cmd := exec.Command(c.binaryPath, args...)
	output, err := cmd.Output()

	if quiet {
		// In quiet mode, exit 0 = running, exit 1 = exited
		result := &StatusResult{Running: err == nil}
		if err != nil {
			// Parse exit code from output
			exitCodeStr := strings.TrimSpace(string(output))
			if exitCode, parseErr := strconv.Atoi(exitCodeStr); parseErr == nil {
				result.ExitCode = &exitCode
			}
		}
		return result, nil
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("interminai status failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("interminai status failed: %w", err)
	}

	return parseStatusOutput(string(output))
}

// parseStatusOutput parses status command output.
func parseStatusOutput(output string) (*StatusResult, error) {
	result := &StatusResult{}
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Running:") {
			result.Running = strings.TrimSpace(strings.TrimPrefix(line, "Running:")) == "true"
		} else if strings.HasPrefix(line, "Activity:") {
			result.Activity = strings.TrimSpace(strings.TrimPrefix(line, "Activity:")) == "true"
		} else if strings.HasPrefix(line, "Exit code:") {
			exitCodeStr := strings.TrimSpace(strings.TrimPrefix(line, "Exit code:"))
			if exitCode, err := strconv.Atoi(exitCodeStr); err == nil {
				result.ExitCode = &exitCode
			}
		}
	}

	return result, nil
}

// Wait waits for activity or exit.
func (c *Client) Wait(socketPath string, quiet bool, timeoutMs int) (*WaitResult, error) {
	args := []string{"wait", "--socket", socketPath}
	if quiet {
		args = append(args, "--quiet")
	}

	cmd := exec.Command(c.binaryPath, args...)
	output, err := cmd.Output()

	if quiet {
		result := &WaitResult{Exited: true}
		exitCodeStr := strings.TrimSpace(string(output))
		if exitCode, parseErr := strconv.Atoi(exitCodeStr); parseErr == nil {
			result.ExitCode = &exitCode
		}
		return result, nil
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("interminai wait failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("interminai wait failed: %w", err)
	}

	return parseWaitOutput(string(output))
}

// parseWaitOutput parses wait command output.
func parseWaitOutput(output string) (*WaitResult, error) {
	result := &WaitResult{}
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Terminal activity:") {
			result.Activity = strings.TrimSpace(strings.TrimPrefix(line, "Terminal activity:")) == "true"
		} else if strings.HasPrefix(line, "Application exited:") {
			result.Exited = strings.TrimSpace(strings.TrimPrefix(line, "Application exited:")) == "true"
		}
	}

	return result, nil
}

// Resize resizes the terminal.
func (c *Client) Resize(socketPath string, cols, rows int) error {
	args := []string{"resize", "--socket", socketPath, "--size", fmt.Sprintf("%dx%d", cols, rows)}
	cmd := exec.Command(c.binaryPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("interminai resize failed: %s", string(output))
	}
	return nil
}

// Kill sends a signal to the process.
func (c *Client) Kill(socketPath string, signal string) error {
	args := []string{"kill", "--socket", socketPath, "--signal", signal}
	cmd := exec.Command(c.binaryPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("interminai kill failed: %s", string(output))
	}
	return nil
}

// Stop stops the session.
func (c *Client) Stop(socketPath string) error {
	args := []string{"stop", "--socket", socketPath}
	cmd := exec.Command(c.binaryPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("interminai stop failed: %s", string(output))
	}
	return nil
}

// Debug gets debug information.
func (c *Client) Debug(socketPath string, clear bool) (*DebugResult, error) {
	args := []string{"debug", "--socket", socketPath}
	if clear {
		args = append(args, "--clear")
	}

	cmd := exec.Command(c.binaryPath, args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("interminai debug failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("interminai debug failed: %w", err)
	}

	return parseDebugOutput(string(output))
}

// parseDebugOutput parses debug command output.
func parseDebugOutput(output string) (*DebugResult, error) {
	result := &DebugResult{
		UnhandledEscapeSequences: []string{},
		Termios:                  &TermiosInfo{},
	}

	scanner := bufio.NewScanner(strings.NewReader(output))
	inUnhandled := false
	inTermios := false

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "Unhandled escape sequences:") {
			inUnhandled = true
			inTermios = false
			continue
		} else if strings.HasPrefix(line, "Termios:") {
			inUnhandled = false
			inTermios = true
			continue
		}

		if inUnhandled && strings.HasPrefix(line, "  ") {
			result.UnhandledEscapeSequences = append(result.UnhandledEscapeSequences, strings.TrimSpace(line))
		} else if inTermios && strings.HasPrefix(line, "  ") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "Mode:") {
				result.Termios.Mode = strings.TrimSpace(strings.TrimPrefix(trimmed, "Mode:"))
			} else if strings.HasPrefix(trimmed, "Flags:") {
				flags := strings.TrimSpace(strings.TrimPrefix(trimmed, "Flags:"))
				result.Termios.Flags = strings.Fields(flags)
			}
		}
	}

	return result, nil
}
