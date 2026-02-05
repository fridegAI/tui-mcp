# TUI MCP Server

An MCP (Model Context Protocol) server that enables AI agents to interact with terminal-based applications through [interminai](https://github.com/guibef/interminai-plus).

## Features

- **10 MCP tools** for complete TUI control: start, input, output, status, wait, resize, signal, stop, debug, observe
- **Session management** with automatic inactivity timeout cleanup
- **Security** with configurable command allowlist
- **Escape sequence support** for arrow keys, function keys, and special characters

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

### Claude Desktop

Add to `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "tui": {
      "command": "/path/to/tui-mcp"
    }
  }
}
```

### Cline / Other MCP Clients

Configure the MCP server command as: `/path/to/tui-mcp`

## MCP Tools

| Tool | Description |
|------|-------------|
| `tui.start` | Start a TUI session with a command |
| `tui.input` | Send keystrokes (supports `\r`, `\e`, `\e[A`, etc.) |
| `tui.output` | Get current terminal screen content |
| `tui.status` | Check if session is running and has activity |
| `tui.wait` | Wait for terminal activity or process exit |
| `tui.resize_screen` | Resize terminal (sends SIGWINCH) |
| `tui.signal` | Send signal (SIGINT, SIGTERM, etc.) |
| `tui.stop` | Stop and cleanup session |
| `tui.debug` | Get termios mode/flags for escape sequence handling |
| `tui.observe` | Combined output + status in one call |

## Usage Example

```
Agent: Start vim to edit a file
→ tui.start {"command": "vim test.txt"}
← {"session_id": "/tmp/interminai-xxx/socket", "pid": 12345}

Agent: Type some text
→ tui.input {"session_id": "...", "text": "iHello World"}

Agent: Save and quit
→ tui.input {"session_id": "...", "text": "\e:wq\r"}

Agent: Cleanup
→ tui.stop {"session_id": "..."}
```

## Escape Sequences

| Key | Sequence |
|-----|----------|
| Enter | `\r` |
| Escape | `\e` |
| Tab | `\t` |
| Arrow Up | `\e[A` |
| Arrow Down | `\e[B` |
| Arrow Right | `\e[C` |
| Arrow Left | `\e[D` |
| Ctrl+C | Use `tui.signal` with `SIGINT` |

## Security

The server includes a command allowlist that blocks dangerous patterns like `rm -rf` and `mkfs`. Default allowed commands include: `ls`, `cat`, `grep`, `vim`, `git`, `htop`, `python`, `node`, etc.

## Development

```bash
# Run tests
go test -v ./...

# Build
go build -o tui-mcp ./cmd/tui-mcp
```

## License

MIT
