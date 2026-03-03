package permission

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Action represents a permission request.
type Action struct {
	Tool   string
	Action string
	Path   string
	Params map[string]interface{}
}

// RiskLevel represents the danger level of an action.
type RiskLevel int

const (
	RiskLow RiskLevel = iota
	RiskMedium
	RiskHigh
	RiskCritical
)

// Result is the outcome of a permission check.
type Result struct {
	Allowed bool
	Message string
	Reason  string
}

// Service manages tool permissions.
type Service struct {
	skipConfirmation bool
	allowedTools     map[string]bool
	blockedTools     map[string]bool
	allowedPaths     []string
	blockedPaths     []string

	// Session-based approvals
	sessionApprovals map[string]bool
	mu               sync.RWMutex
}

// NewService creates a new permission service.
func NewService(skip bool) *Service {
	return &Service{
		skipConfirmation: skip,
		allowedTools:     make(map[string]bool),
		blockedTools:     make(map[string]bool),
		sessionApprovals: make(map[string]bool),
		allowedPaths:     []string{},
		blockedPaths:     []string{},
	}
}

// Configure sets up the permission service.
func (s *Service) Configure(allowed, blocked []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, tool := range allowed {
		s.allowedTools[tool] = true
	}
	for _, tool := range blocked {
		s.blockedTools[tool] = true
	}
}

// SetAllowedPaths sets allowed path patterns.
func (s *Service) SetAllowedPaths(paths []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.allowedPaths = paths
}

// SetBlockedPaths sets blocked path patterns.
func (s *Service) SetBlockedPaths(paths []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.blockedPaths = paths
}

// Check evaluates a permission request.
func (s *Service) Check(action Action) Result {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if tool is explicitly blocked
	if s.blockedTools[action.Tool] {
		return Result{
			Allowed: false,
			Message: fmt.Sprintf("Tool '%s' is blocked", action.Tool),
			Reason:  "blocked",
		}
	}

	// Check if only specific tools are allowed
	if len(s.allowedTools) > 0 {
		if !s.allowedTools[action.Tool] {
			return Result{
				Allowed: false,
				Message: fmt.Sprintf("Tool '%s' is not in allowed list", action.Tool),
				Reason:  "not_allowed",
			}
		}
	}

	// Check path restrictions
	if action.Path != "" {
		if !s.isPathAllowed(action.Path) {
			return Result{
				Allowed: false,
				Message: fmt.Sprintf("Path '%s' is not allowed", action.Path),
				Reason:  "path_blocked",
			}
		}
	}

	// Assess risk level
	risk := s.assessRisk(action)

	// Skip confirmation for low risk if configured
	if s.skipConfirmation && risk < RiskHigh {
		return Result{Allowed: true}
	}

	// Check session approval for high risk
	if risk >= RiskHigh {
		key := s.actionKey(action)
		if !s.sessionApprovals[key] {
			return Result{
				Allowed: false,
				Message: s.buildPrompt(action, risk),
				Reason:  "needs_approval",
			}
		}
	}

	return Result{Allowed: true}
}

// Approve grants permission for an action.
func (s *Service) Approve(action Action) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessionApprovals[s.actionKey(action)] = true
}

// ApproveTool grants permission for all uses of a tool.
func (s *Service) ApproveTool(tool string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessionApprovals["tool:"+tool] = true
}

// isPathAllowed checks if a path is allowed.
func (s *Service) isPathAllowed(path string) bool {
	// Check blocked paths
	for _, blocked := range s.blockedPaths {
		if strings.HasPrefix(path, blocked) {
			return false
		}
	}

	// If allowed paths specified, must match one
	if len(s.allowedPaths) > 0 {
		for _, allowed := range s.allowedPaths {
			if strings.HasPrefix(path, allowed) {
				return true
			}
		}
		return false
	}

	return true
}

// assessRisk determines the risk level of an action.
func (s *Service) assessRisk(action Action) RiskLevel {
	switch action.Tool {
	case "fs_read", "fs_glob", "fs_grep":
		return RiskLow

	case "fs_write":
		// Check if overwriting existing file
		if action.Path != "" {
			if _, err := os.Stat(action.Path); err == nil {
				return RiskHigh // Overwrite
			}
		}
		return RiskMedium

	case "fs_edit":
		return RiskHigh

	case "shell_exec":
		cmd := ""
		if c, ok := action.Params["command"].(string); ok {
			cmd = c
		}

		// Check for dangerous commands
		dangerous := []string{
			"rm -rf", ":(){ :|: & };:", "mkfs", "dd if=/dev/zero",
			"> /dev/sda", "curl.*|.*sh", "wget.*|.*sh",
		}
		lowerCmd := strings.ToLower(cmd)
		for _, d := range dangerous {
			if strings.Contains(lowerCmd, d) {
				return RiskCritical
			}
		}

		// Check for destructive operations
		destructive := []string{"rm ", "rmdir ", "unlink ", "truncate "}
		for _, d := range destructive {
			if strings.HasPrefix(lowerCmd, d) {
				return RiskHigh
			}
		}

		return RiskMedium

	default:
		return RiskMedium
	}
}

// buildPrompt creates a user prompt for approval.
func (s *Service) buildPrompt(action Action, risk RiskLevel) string {
	riskStr := ""
	switch risk {
	case RiskMedium:
		riskStr = "⚠️ Medium Risk"
	case RiskHigh:
		riskStr = "🔴 High Risk"
	case RiskCritical:
		riskStr = "☠️ CRITICAL"
	}

	prompt := fmt.Sprintf("%s\nTool: %s\nAction: %s", riskStr, action.Tool, action.Action)
	if action.Path != "" {
		prompt += fmt.Sprintf("\nPath: %s", action.Path)
	}
	prompt += "\n\nAllow this action? (yes/no/always)"

	return prompt
}

// actionKey generates a unique key for an action.
func (s *Service) actionKey(action Action) string {
	return fmt.Sprintf("%s:%s:%s", action.Tool, action.Action, action.Path)
}

// GetRiskDescription returns a human-readable risk description.
func (s *Service) GetRiskDescription(risk RiskLevel) string {
	switch risk {
	case RiskLow:
		return "Low - Read-only operations"
	case RiskMedium:
		return "Medium - Write operations"
	case RiskHigh:
		return "High - Destructive operations"
	case RiskCritical:
		return "CRITICAL - System-threatening"
	default:
		return "Unknown"
	}
}

// DefaultAllowedPaths returns safe default paths.
func DefaultAllowedPaths(workDir string) []string {
	return []string{
		workDir,
		os.TempDir(),
	}
}

// DefaultBlockedPaths returns paths that should never be touched.
func DefaultBlockedPaths() []string {
	home, _ := os.UserHomeDir()
	return []string{
		"/etc",
		"/usr/bin",
		"/bin",
		"/sbin",
		"/boot",
		filepath.Join(home, ".ssh"),
		filepath.Join(home, ".gnupg"),
	}
}
