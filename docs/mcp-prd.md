# 📘 Design Doc
## Thin MCP, Smart Agent — TUI Automation via MCP

**Status:** Implementation-ready
**Audience:** Engineers building AI agents (e.g. Cline) that must interact with existing TUI / interactive CLI tools.

---

## 1. Problem Statement

Many DevOps and platform workflows rely on **interactive Terminal User Interfaces (TUIs)**:
- arrow-key menus (`dialog`, `whiptail`)
- curses-based full-screen apps
- interactive Bash / Python scripts
- confirmation prompts and selection flows

These tools:
- cannot easily be refactored into APIs
- must run unchanged
- are currently human-operated

**Goal:**
Enable an agent like **Cline** to operate these TUIs autonomously via **MCP**, while keeping reasoning, planning, and intent entirely inside the agent.

---

## 2. Architectural Principle

### Thin MCP, Smart Agent

**MCP exposes terminal capabilities and UI state.
Cline decides what to do.**

- MCP server = TUI execution + observation
- Agent (Cline) = reasoning, intent, safety, recovery

The MCP server must never:
- plan tasks
- choose UI actions
- infer user intent

---

## 3. High-Level Architecture

```

┌──────────────┐
│    Cline     │
│  (LLM Agent) │
└──────┬───────┘
       │ MCP (JSON-RPC)
┌──────▼────────────────────┐
│ MCP TUI Server            │
│                           │
│  Interminai               │  ← PTY proxy
│      │                    │
│      ▼                    │
│  Raw Terminal Buffer      │
│                           │
└──────┬────────────────────┘
       │ PTY
┌──────▼───────┐
│ Real TUI App │
│ (unchanged)  │
└──────────────┘

```

---

## 4. Core Technology Choice: Interminai-plus

We use [**interminai-plus**](https://github.com/guibef/interminai-plus), a forked version of **interminai** that has been enhanced with bug fixes and improvements.

### Key Advantages:
- **Single Daemon**: No need to manage multiple background processes.
- **Robust PTY**: Leverages **interminai-plus**'s kernel-grade PTY management and socket-based isolation.
- **Bug Fixes**: Includes critical fixes for terminal emulation and character handling.
- **Unified Protocol**: A single socket API provides raw character grids.

---

## 5. Non-Goals

The MCP server explicitly does **not**:
- embed an LLM
- decide UI actions
- perform task planning
- encode business logic
- batch or optimize agent decisions

---

## 6. Session Lifecycle & State

Interactive executions are stateful sessions managed by the underlying drivers.

### Lifecycle Phases
1. **Creation**: `tui.start` initializes a session.
    - If `venv_path` is provided, the MCP server automatically sets `VIRTUAL_ENV` and updates `PATH` to use the specified Python virtual environment before spawning the process.
    - The MCP server starts an **interminai-plus** session via its Unix socket API.
2. **Active**: Agent issues `tui.input` and `tui.output` calls.
3. **Termination**: Triggered by process exit, `tui.stop`, or inactivity timeout.
    - Clean up via **interminai-plus** stop commands.

### State Persistence
- **In-Memory**: The MCP server tracks active `session_id` to socket session mappings.
- **Cleanup**: On termination, the server ensures PTYs are killed and socket files are removed.

---

## 7. Screen & UI Representation

### Raw Terminal Output (`tui.output`)
The primary observation method returns structured terminal state:
```json
{
  "content": "string",
  "rows": "number",
  "cols": "number",
  "cursor": {
    "row": "number (1-based, optional)",
    "col": "number (1-based, optional)"
  }
}
```

- **Content**: Raw terminal output with ANSI escapes preserved
- **Cursor**: Structured cursor position when available (not embedded in text)
- **Dimensions**: Current terminal rows/cols

---

## 8. MCP Tool API

The MCP Tool API follows a **lossless projection** of interminai's command surface into MCP tools. No semantics are added, no behavior is hidden in prompts.

### 8.1 `tui.start`
Start a TUI session.

**Schema:** (maps to `interminai start`)
```json
{
  "command": "string",
  "cwd": "string (optional)",
  "env": { "KEY": "VALUE" },
  "cols": 80,
  "rows": 24,
  "emulator": "xterm | custom",
  "daemon": true,
  "pty_dump_path": "string (optional)"
}
```

**Returns:**
```json
{
  "session_id": "string",
  "socket_path": "string",
  "pid": "number",
  "auto_socket": true
}
```

**Notes:**
- `daemon=true` is the **default**
- `emulator=xterm` should be the MCP default (color + cursor correctness)
- `pty_dump_path` exposes `--pty-dump` safely (debug only)

### 8.2 `tui.input`
Send input to a TUI session.

**Schema:** (maps to `interminai input`)
```json
{
  "session_id": "string",
  "text": "string",
  "password": false
}
```

**Escape sequences supported:**
- `text` supports **full C-style escapes**
- Arrow keys, Ctrl keys, function keys are passed **verbatim**
- MCP does **not** invent key abstractions — the agent controls escapes
- This keeps parity with interminai and avoids leaky abstractions

### 8.3 `tui.output`
Get the current terminal output.

**Schema:** (maps to `interminai output`)
```json
{
  "session_id": "string",
  "color": true | false,
  "cursor": "none | print | inverse | both"
}
```

**Returns:**
```json
{
  "content": "string",
  "rows": "number",
  "cols": "number",
  "cursor": {
    "row": "number (1-based, optional)",
    "col": "number (1-based, optional)"
  }
}
```

**Design decision:**
- Cursor is **structured when available**, not embedded in text
- Color output is preserved verbatim (ANSI escapes)

### 8.4 `tui.status`
Check session status.

**Schema:** (maps to `interminai status`)
```json
{
  "session_id": "string",
  "quiet": false
}
```

**Returns:**
```json
{
  "running": true,
  "activity": true,
  "exit_code": null
}
```

**Notes:**
- `quiet=true` maps to exit-code-only behavior
- Activity flag is important for agent polling loops

### 8.5 `tui.wait`
Wait for session state change.

**Schema:** (maps to `interminai wait`)
```json
{
  "session_id": "string",
  "mode": "activity | exit",
  "timeout_ms": "number (optional)"
}
```

**Returns:**
```json
{
  "activity": true,
  "exited": false,
  "exit_code": null
}
```

**Design improvement:**
- Replaces `--quiet` with explicit `mode`
- Much easier for an agent to reason about

### 8.6 `tui.resize_screen`
Resize terminal window.

**Schema:** (maps to `interminai resize`)
```json
{
  "session_id": "string",
  "cols": "number",
  "rows": "number"
}
```

**Behavior:**
- Sends `SIGWINCH`
- Agent can deliberately trigger redraws (critical for TUIs)

### 8.7 `tui.signal`
Send signal to process.

**Schema:** (maps to `interminai kill`)
```json
{
  "session_id": "string",
  "signal": "SIGINT | SIGTERM | SIGKILL | SIGQUIT | SIGUSR1 | number"
}
```

**Notes:**
- Numeric and named signals supported
- `SIGINT` is the canonical Ctrl+C

### 8.8 `tui.stop`
Stop and clean up session.

**Schema:** (maps to `interminai stop`)
```json
{
  "session_id": "string"
}
```

**Contract:**
- **Always call** for cleanup
- MCP guarantees socket cleanup if auto-generated

### 8.9 `tui.debug`
Get debugging information.

**Schema:** (maps to `interminai debug`)
```json
{
  "session_id": "string",
  "clear": false
}
```

**Returns:**
```json
{
  "unhandled_escape_sequences": ["string"],
  "termios": {
    "mode": "raw | cooked",
    "flags": ["ECHO", "ISIG", "ICRNL", "IXON", "OPOST"],
    "hex": {
      "iflag": "string",
      "oflag": "string",
      "lflag": "string",
      "cflag": "string"
    },
    "control_chars": {
      "VINTR": "^C",
      "VEOF": "^D"
    }
  }
}
```

**Why this matters:**
- Lets the agent **diagnose broken TUIs**
- Explains `\r` vs `\n` bugs
- Essential for robust automation

### 8.10 Optional MCP-level Helpers
*(Not interminai-plus commands, but safe abstractions)*

**`tui.observe`** - Thin wrapper around `output + status`:
```json
{
  "session_id": "string"
}
```
**Returns:**
```json
{
  "screen": "...",
  "running": true,
  "activity": true
}
```
This is a **convenience**, not a capability addition.

**`tui.ensure_started`** - Useful for crash recovery:
```json
{
  "session_id": "string"
}
```
Returns whether socket/process still exists.

---

## 9. Control Loop (Agent Side)

Cline runs the full reasoning loop:
`tui.output → reason → decide → tui.input → tui.wait → tui.output`

The agent leverages structured cursor information to avoid complex coordinate-based math or fragile regex matching.

---

## 10. Action Granularity Rules

- **One MCP call = one mechanical action** (key press, text input, resize)
- **Agent manages sequencing** using `tui.wait` for synchronization
- **No complex multi-key macros** in a single `tui.input` call
- Use `tui.wait(mode="activity")` to wait for UI updates before next observation
- Use `tui.wait(mode="exit")` to wait for process termination

---

## 11. Safety & Guardrails

### Execution Security
- **Command Allowlisting**: Only approved binaries (e.g., `git`, `htop`, `vim`, `apt`) can be executed.
- **Path Restrictions**: `cwd` must be within a safe, configurable directory tree.
- **Environment Scrubbing**: Sensitive host environment variables (e.g., `GITHUB_TOKEN`) are filtered.

### Resource Limits
- **Inactivity Timeout**: Sessions automatically close after 300 seconds of no agent activity.
- **Zombie Prevention**: The MCP server ensures that PTYs are killed even if the client disconnects.

---

## 12. Error Handling

### Technical Edge Cases
- **PTY Disconnect**: If the PTY dies, the session status is updated to `exited`.
- **Input Lag**: The `tui.input` tool waits for proxy confirmation before returning.

---

## 13. Observability & Debugging

- **Live Preview**: The **interminai-plus** proxy can provide a live view of the terminal for manual monitoring.
- **MCP Logging**: Raw ANSI logs are sent to the client via `notifications/message`.

---

## 14. Summary

This design turns **interactive TUIs into first-class MCP tools** by leveraging **interminai-plus**, ensuring that **Cline remains the intelligent decision-maker** in the loop.
