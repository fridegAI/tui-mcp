package main

import (
	"log"
	"os"
	"os/signal"
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
	// Create session manager
	manager := session.NewManager(SessionTimeout)

	// Create command allowlist (enabled by default)
	allowlist := security.NewAllowlist(true)

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
