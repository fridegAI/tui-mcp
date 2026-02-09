package main

import (
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/fridegai/tui-mcp/internal/security"
	"github.com/fridegai/tui-mcp/internal/session"
	"github.com/fridegai/tui-mcp/internal/tools"
	"github.com/mark3labs/mcp-go/server"
)

const (
	ServerName     = "tui-mcp"
	ServerVersion  = "0.1.1"
	SessionTimeout = 300 * time.Second // 5 minutes
)

func main() {
	// Parse configuration
	sessionTimeout := SessionTimeout
	if timeoutStr := os.Getenv("SESSION_TIMEOUT"); timeoutStr != "" {
		if d, err := time.ParseDuration(timeoutStr); err == nil {
			sessionTimeout = d
		} else {
			log.Printf("Invalid SESSION_TIMEOUT format: %v, using default", err)
		}
	}

	// Create session manager
	manager, err := session.NewManager(sessionTimeout)
	if err != nil {
		log.Fatalf("Failed to create session manager: %v", err)
	}

	// Create command allowlist (enabled by default)
	allowlist := security.NewAllowlist(true)

	// Add configured allowed commands
	if allowedCmds := os.Getenv("ALLOWED_COMMANDS"); allowedCmds != "" {
		for _, cmd := range strings.Split(allowedCmds, ",") {
			if trimmed := strings.TrimSpace(cmd); trimmed != "" {
				allowlist.AddAllowed(trimmed)
			}
		}
	}

	// Add configured blocked patterns
	if blockedPatterns := os.Getenv("BLOCKED_PATTERNS"); blockedPatterns != "" {
		for _, pattern := range strings.Split(blockedPatterns, ",") {
			if trimmed := strings.TrimSpace(pattern); trimmed != "" {
				allowlist.AddBlocked(trimmed)
			}
		}
	}

	// Create MCP server
	s := server.NewMCPServer(
		ServerName,
		ServerVersion,
		server.WithToolCapabilities(true),
	)

	// Register tools
	handler := tools.NewHandler(manager, allowlist)
	handler.RegisterTools(s)

	// Handle shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("Shutting down...")
		manager.Stop()
		os.Exit(0)
	}()

	// Run server with stdio transport
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
