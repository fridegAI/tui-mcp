package security

import (
	"testing"
)

func TestAllowlist_Validate_AllowedCommands(t *testing.T) {
	allowlist := NewAllowlist(true)

	tests := []struct {
		command string
		allowed bool
	}{
		{"git status", true},
		{"vim file.txt", true},
		{"htop", true},
		{"tail -f log.txt", true},
		{"head -n 10 file.txt", true},
		{"man ls", true},
		{"ping google.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			ok, reason := allowlist.Validate(tt.command)
			if ok != tt.allowed {
				t.Errorf("Validate(%q) = %v, want %v (reason: %s)", tt.command, ok, tt.allowed, reason)
			}
		})
	}
}

func TestAllowlist_Validate_BlockedPatterns(t *testing.T) {
	allowlist := NewAllowlist(true)

	tests := []struct {
		command string
		blocked bool
	}{
		{"rm -rf /", true},
		{"rm -fr /home", true},
		{"rm file.txt", true}, // rm is not in allowlist, so it's blocked
		{"mkfs.ext4 /dev/sda", true},
		{"dd if=/dev/zero of=/dev/sda", true},
		{"chmod -R 777 /", true},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			ok, _ := allowlist.Validate(tt.command)
			if ok == tt.blocked {
				t.Errorf("Validate(%q) should be blocked=%v", tt.command, tt.blocked)
			}
		})
	}
}

func TestAllowlist_Validate_DisabledAllowlist(t *testing.T) {
	allowlist := NewAllowlist(false) // Disabled

	// Should allow any command except blocked patterns
	ok, _ := allowlist.Validate("some_unknown_command arg1 arg2")
	if !ok {
		t.Error("Disabled allowlist should allow unknown commands")
	}

	// But still block dangerous patterns
	ok, _ = allowlist.Validate("rm -rf /")
	if ok {
		t.Error("Disabled allowlist should still block dangerous patterns")
	}
}

func TestExtractBaseCommand(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ls -la", "ls"},
		{"cat file.txt", "cat"},
		{"GIT_EDITOR=vim git commit", "git"},
		{"sudo apt install", "apt"},
		{"/usr/bin/vim file.txt", "vim"},
		{"env VAR=value python script.py", "python"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractBaseCommand(tt.input)
			if result != tt.expected {
				t.Errorf("extractBaseCommand(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAllowlist_AddRemove(t *testing.T) {
	allowlist := NewAllowlist(true)

	// Custom command should be blocked initially
	ok, _ := allowlist.Validate("myapp --flag")
	if ok {
		t.Error("myapp should not be allowed initially")
	}

	// Add to allowlist
	allowlist.AddAllowed("myapp")
	ok, _ = allowlist.Validate("myapp --flag")
	if !ok {
		t.Error("myapp should be allowed after AddAllowed")
	}

	// Remove from allowlist
	allowlist.RemoveAllowed("myapp")
	ok, _ = allowlist.Validate("myapp --flag")
	if ok {
		t.Error("myapp should not be allowed after RemoveAllowed")
	}
}
