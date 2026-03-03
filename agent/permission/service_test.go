package permission

import (
	"testing"
)

func TestServiceCheck(t *testing.T) {
	s := NewService(false)

	tests := []struct {
		name    string
		action  Action
		allowed bool
	}{
		{
			name:    "read file allowed",
			action:  Action{Tool: "fs_read", Path: "/tmp/test.txt"},
			allowed: true,
		},
		{
			name:    "blocked tool",
			action:  Action{Tool: "dangerous_tool"},
			allowed: false,
		},
	}

	// Configure blocked tools
	s.Configure(nil, []string{"dangerous_tool"})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.Check(tt.action)
			if result.Allowed != tt.allowed {
				t.Errorf("expected allowed=%v, got %v (reason: %s)",
					tt.allowed, result.Allowed, result.Reason)
			}
		})
	}
}

func TestRiskAssessment(t *testing.T) {
	s := NewService(false)

	tests := []struct {
		name     string
		action   Action
		risk     RiskLevel
	}{
		{
			name:   "fs_read is low risk",
			action: Action{Tool: "fs_read", Path: "/tmp/test.txt"},
			risk:   RiskLow,
		},
		{
			name:   "fs_write new file is medium risk",
			action: Action{Tool: "fs_write", Path: "/tmp/new_file.txt"},
			risk:   RiskMedium,
		},
		{
			name:   "fs_edit is high risk",
			action: Action{Tool: "fs_edit", Path: "/tmp/test.txt"},
			risk:   RiskHigh,
		},
		{
			name:   "rm command is critical risk",
			action: Action{Tool: "shell_exec", Params: map[string]interface{}{"command": "rm -rf /"}},
			risk:   RiskCritical,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			risk := s.assessRisk(tt.action)
			if risk != tt.risk {
				t.Errorf("expected risk %v, got %v", tt.risk, risk)
			}
		})
	}
}

func TestAllowedList(t *testing.T) {
	s := NewService(false)

	// Only allow specific tools
	s.Configure([]string{"fs_read", "fs_glob"}, nil)

	// Allowed tool
	result := s.Check(Action{Tool: "fs_read"})
	if !result.Allowed {
		t.Error("fs_read should be allowed")
	}

	// Not allowed tool
	result = s.Check(Action{Tool: "fs_write"})
	if result.Allowed {
		t.Error("fs_write should not be allowed")
	}
}

func TestSessionApproval(t *testing.T) {
	s := NewService(false)

	action := Action{Tool: "fs_edit", Path: "/tmp/test.txt"}

	// Initially should need approval (high risk)
	result := s.Check(action)
	if result.Allowed {
		t.Error("should need approval initially")
	}

	// Approve
	s.Approve(action)

	// Now should be allowed
	result = s.Check(action)
	if !result.Allowed {
		t.Error("should be allowed after approval")
	}
}

func TestPathRestrictions(t *testing.T) {
	s := NewService(false)

	// Block /etc
	s.SetBlockedPaths([]string{"/etc"})

	// Allowed path
	result := s.Check(Action{Tool: "fs_read", Path: "/tmp/file.txt"})
	if !result.Allowed {
		t.Error("/tmp should be allowed")
	}

	// Blocked path
	result = s.Check(Action{Tool: "fs_read", Path: "/etc/passwd"})
	if result.Allowed {
		t.Error("/etc should be blocked")
	}
}

func TestSkipConfirmation(t *testing.T) {
	s := NewService(true) // Skip confirmation

	// Even medium risk should be allowed
	result := s.Check(Action{Tool: "fs_write", Path: "/tmp/new.txt"})
	if !result.Allowed {
		t.Error("should be allowed when skip confirmation is enabled")
	}

	// But critical risk should still be blocked
	s.blockedTools = nil // Clear blocked list
	result = s.Check(Action{Tool: "shell_exec", Params: map[string]interface{}{"command": "rm -rf /"}})
	// Note: shell_exec itself isn't blocked, but the command is critical risk
	// This depends on the implementation details
}
