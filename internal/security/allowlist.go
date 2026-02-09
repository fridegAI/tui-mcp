package security

import (
	"strings"
)

// DefaultAllowed contains safe READ commands.
var DefaultAllowed = []string{
	"head", "tail", "less", "more",
	"vim", "nvim", "vi", "nano", "emacs", "git",
	"htop", "top", "man",
	"netstat", "ping", "traceroute", "nc",
}

// BlockedPatterns contains dangerous command patterns.
var BlockedPatterns = []string{
	"rm -rf",
	"rm -fr",
	"rm -r /",
	"rm -f /",
	"mkfs",
	"dd if=",
	"> /dev/",
	"chmod -R 777",
	"chown -R",
}

// Allowlist validates commands against security policies.
type Allowlist struct {
	allowed map[string]bool
	blocked []string
	enabled bool
}

// NewAllowlist creates a new command allowlist.
func NewAllowlist(enabled bool) *Allowlist {
	allowed := make(map[string]bool)
	for _, cmd := range DefaultAllowed {
		allowed[cmd] = true
	}

	return &Allowlist{
		allowed: allowed,
		blocked: BlockedPatterns,
		enabled: enabled,
	}
}

// Validate checks if a command is allowed.
func (a *Allowlist) Validate(command string) (bool, string) {
	// Always check blocked patterns
	for _, pattern := range a.blocked {
		if strings.Contains(command, pattern) {
			return false, "command contains blocked pattern: " + pattern
		}
	}

	// If allowlist is disabled, allow everything else
	if !a.enabled {
		return true, ""
	}

	// Extract base command
	baseCmd := extractBaseCommand(command)
	if baseCmd == "" {
		return false, "could not extract base command"
	}

	// Check allowlist
	if !a.allowed[baseCmd] {
		return false, "command not in allowlist: " + baseCmd
	}

	return true, ""
}

// AddAllowed adds a command to the allowlist.
func (a *Allowlist) AddAllowed(cmd string) {
	a.allowed[cmd] = true
}

// RemoveAllowed removes a command from the allowlist.
func (a *Allowlist) RemoveAllowed(cmd string) {
	delete(a.allowed, cmd)
}

// AddBlocked adds a pattern to the blocked list.
func (a *Allowlist) AddBlocked(pattern string) {
	a.blocked = append(a.blocked, pattern)
}

// extractBaseCommand extracts the base command from a command string.
func extractBaseCommand(command string) string {
	// Handle env var assignments at the start
	parts := strings.Fields(command)
	for _, part := range parts {
		// Skip env var assignments (KEY=value)
		if strings.Contains(part, "=") && !strings.HasPrefix(part, "-") {
			continue
		}
		// Skip common prefixes
		if part == "sudo" || part == "env" || part == "time" || part == "nice" {
			continue
		}
		// This is the base command
		// Handle paths like /usr/bin/vim
		if strings.Contains(part, "/") {
			parts := strings.Split(part, "/")
			return parts[len(parts)-1]
		}
		return part
	}
	return ""
}
