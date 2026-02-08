# Tasks from improvement

## main.go

- [ ] `SessionTimeout` should be configurable via MCP Server Config, default to 5 minutes

## client.go

- [ ] `BinaryPath` should be reade from evalute from `which interminai`

## allowlist.go

- [ ] `DefaultAllowed` should be configurable via MCP Server Config, use configured allowed commands should be appended to default allowed commands `DefaultAllowed`

- [ ] `BlockedPatterns` should be configurable via MCP Server Config, use configured blocked patterns should be appended to default blocked patterns `BlockedPatterns`

## handler.go

- [ ] `tui.start` default emulator should be `xterm`

- [ ] `tui.output` cursor mode default should be `print`
