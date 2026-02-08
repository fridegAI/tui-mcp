# TUI MCP Server

An MCP (Model Context Protocol) server that enables AI agents to interact with terminal-based applications through [interminai](https://github.com/guibef/interminai-plus).

## Features

- **10 MCP tools** for complete TUI control: start, input, output, status, wait, resize, signal, stop, debug, observe
- **Session management** with automatic inactivity timeout cleanup
- **Security** with configurable command allowlist and blocked patterns
- **Escape sequence support** for arrow keys, function keys, and special characters
- **Configurable** session timeouts and security policies via environment variables

## Prerequisites

- Go 1.21+
- [interminai](https://github.com/guibef/interminai-plus) installed and available in PATH

## Installation

```bash
# Clone the repository
git clone git@github.com:fridegAI/tui-mcp.git
cd tui-mcp

# Build
go build -o tui-mcp ./cmd/tui-mcp

# Or install globally
go install ./cmd/tui-mcp
```

## Configuration

The server can be configured using environment variables. These can be set in the MCP client configuration.

| Variable | Description | Default |
|----------|-------------|---------|
| `SESSION_TIMEOUT` | Session inactivity timeout (e.g., "5m", "1h") | `5m` |
| `ALLOWED_COMMANDS` | Comma-separated list of additional allowed commands | (Empty) |
| `BLOCKED_PATTERNS` | Comma-separated list of additional blocked patterns | (Empty) |

### MCP Client Configuration

Make sure to add the environment variables to the `env` object in your MCP client configuration.

**Example (Generic JSON):**

```json
{
  "mcpServers": {
    "tui": {
      "command": "/path/to/tui-mcp",
      "env": {
        "SESSION_TIMEOUT": "10m",
        "ALLOWED_COMMANDS": "kubectl,docker",
        "BLOCKED_PATTERNS": "rm -rf,mkfs"
      }
    }
  }
}
```

## MCP Tools

| Tool | Description | Input Parameters |
|------|-------------|------------------|
| `tui.start` | Start a TUI session | `command` (required), `cols`, `rows`, `cwd`, `env`, `emulator` (default: "xterm") |
| `tui.input` | Send keystrokes | `session_id` (required), `text` (required), `password` |
| `tui.output` | Get screen content | `session_id` (required), `color` (default: false), `cursor` (default: "print") |
| `tui.status` | Check session capability | `session_id` (required), `quiet` |
| `tui.wait` | Wait for activity/exit | `session_id` (required), `mode` ("activity" or "exit"), `timeout_ms` |
| `tui.resize_screen` | Resize terminal | `session_id` (required), `cols` (required), `rows` (required) |
| `tui.signal` | Send signal | `session_id` (required), `signal` (required, e.g. "SIGINT") |
| `tui.stop` | Stop session | `session_id` (required) |
| `tui.debug` | Get debug info | `session_id` (required), `clear` |
| `tui.observe` | Get output + status | `session_id` (required) |

## Usage Example

```
Agent: Start vim to edit a file
→ tui.start {"command": "vim test.txt", "rows": 24, "cols": 80}
← {"session_id": "/tmp/interminai-xxx/socket", "pid": 12345}

Agent: Type some text
→ tui.input {"session_id": "...", "text": "iHello World"}

Agent: Save and quit
→ tui.input {"session_id": "...", "text": "\e:wq\r"}

Agent: Cleanup
→ tui.stop {"session_id": "..."}
```

## Escape Sequences

The `tui.input` tool supports C-style escapes (`\r`, `\n`, `\t`, `\e`) and hex escapes (`\xHH`).

### Arrow Keys
The escape sequence for arrow keys depends on the terminal's Cursor Mode (Normal vs Application).

| Key | Normal Mode | Application Mode |
|-----|-------------|------------------|
| Arrow Up | `\e[A` | `\eOA` |
| Arrow Down | `\e[B` | `\eOB` |
| Arrow Right | `\e[C` | `\eOC` |
| Arrow Left | `\e[D` | `\eOD` |

**Tip:** Use `tui.debug` to inspect the current termios mode if keys are not behaving as expected.

## Security

The server enforces a strict allowlist policy by default.

### Default Policy
- **Allowed Commands:** `ls`, `cat`, `grep`, `vim`, `git`, `htop`, `python`, `node`, `ssh`, and many standard utilities.
- **Blocked Patterns:** Recursive deletion (`rm -rf`, `rm -r /`), formatting (`mkfs`), fork bombs, and device writes (`> /dev/`).

### Customization
You can extend the allowed list or add new blocked patterns via environment variables:
- `ALLOWED_COMMANDS`: Add specific tools required for your workflow (e.g., `kubectl`, `terraform`).
- `BLOCKED_PATTERNS`: Block specific arguments or dangerous command combinations.

## Development

```bash
# Run tests
go test -v ./...

# Build
go build -o tui-mcp ./cmd/tui-mcp
```

### Contributing
1.  Fork the repository
2.  Create your feature branch (`git checkout -b feature/amazing-feature`)
3.  Commit your changes (`git commit -m 'Add some amazing feature'`)
4.  Push to the branch (`git push origin feature/amazing-feature`)
5.  Open a Pull Request

## License

[MIT](LICENSE) License 
