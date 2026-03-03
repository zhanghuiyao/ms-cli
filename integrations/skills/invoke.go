package skills

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Skill represents a callable skill/workflow.
type Skill struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Version     string            `yaml:"version"`
	Steps       []Step            `yaml:"steps"`
	Inputs      map[string]Input  `yaml:"inputs"`
	Outputs     []string          `yaml:"outputs"`
}

// Step represents a single step in a skill.
type Step struct {
	Name    string            `yaml:"name"`
	Tool    string            `yaml:"tool"`
	Command string            `yaml:"command"`  // For shell steps
	Params  map[string]string `yaml:"params"`
	If      string            `yaml:"if"`       // Conditional
	Loop    *LoopConfig       `yaml:"loop"`     // Loop configuration
}

// LoopConfig defines loop parameters.
type LoopConfig struct {
	Over   string `yaml:"over"`   // Variable to iterate over
	As     string `yaml:"as"`     // Loop variable name
	Index  string `yaml:"index"`  // Index variable name
}

// Input defines an input parameter.
type Input struct {
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"`
	Default     string `yaml:"default"`
}

// Repo manages skill repository operations.
type Repo struct {
	URL       string
	Branch    string
	LocalPath string
}

// NewRepo creates a new skill repository manager.
func NewRepo(url, branch, localPath string) *Repo {
	if branch == "" {
		branch = "main"
	}
	return &Repo{
		URL:       url,
		Branch:    branch,
		LocalPath: localPath,
	}
}

// Sync clones or pulls the repository.
func (r *Repo) Sync() error {
	// Check if already cloned
	if _, err := os.Stat(filepath.Join(r.LocalPath, ".git")); err == nil {
		// Pull latest
		cmd := exec.Command("git", "-C", r.LocalPath, "pull", "origin", r.Branch)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git pull failed: %w\n%s", err, output)
		}
		return nil
	}

	// Clone repository
	cmd := exec.Command("git", "clone", "--branch", r.Branch, r.URL, r.LocalPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %w\n%s", err, output)
	}

	return nil
}

// LoadSkill loads a skill from YAML file.
func (r *Repo) LoadSkill(name string) (*Skill, error) {
	skillPath := filepath.Join(r.LocalPath, "skills", name, "skill.yaml")
	data, err := os.ReadFile(skillPath)
	if err != nil {
		return nil, fmt.Errorf("read skill file: %w", err)
	}

	skill := &Skill{}
	// Simple YAML parsing - in production use yaml.Unmarshal
	// For MVP, we'll do basic parsing
	_ = data
	return skill, nil
}

// ListSkills returns all available skills.
func (r *Repo) ListSkills() ([]string, error) {
	skillsDir := filepath.Join(r.LocalPath, "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var skills []string
	for _, entry := range entries {
		if entry.IsDir() {
			skills = append(skills, entry.Name())
		}
	}
	return skills, nil
}

// Invoker executes skill workflows.
type Invoker struct {
	repo     *Repo
	registry interface{} // Tool registry for executing tool steps
}

// NewInvoker creates a new skill invoker.
func NewInvoker(repo *Repo) *Invoker {
	return &Invoker{
		repo: repo,
	}
}

// Invoke executes a skill with given inputs.
func (i *Invoker) Invoke(skillName string, inputs map[string]string) (*Result, error) {
	skill, err := i.repo.LoadSkill(skillName)
	if err != nil {
		return nil, fmt.Errorf("load skill: %w", err)
	}

	result := &Result{
		SkillName: skillName,
		Outputs:   make(map[string]string),
		StepResults: make([]StepResult, 0),
	}

	// Validate inputs
	if err := i.validateInputs(skill, inputs); err != nil {
		return nil, err
	}

	// Merge with defaults
	params := i.mergeInputs(skill, inputs)

	// Execute steps
	for _, step := range skill.Steps {
		stepResult, err := i.executeStep(step, params)
		result.StepResults = append(result.StepResults, stepResult)

		if err != nil {
			result.Success = false
			result.Error = err.Error()
			return result, err
		}
	}

	result.Success = true
	return result, nil
}

// validateInputs checks if all required inputs are provided.
func (i *Invoker) validateInputs(skill *Skill, inputs map[string]string) error {
	for name, input := range skill.Inputs {
		if input.Required {
			if _, ok := inputs[name]; !ok {
				return fmt.Errorf("required input missing: %s", name)
			}
		}
	}
	return nil
}

// mergeInputs merges provided inputs with defaults.
func (i *Invoker) mergeInputs(skill *Skill, inputs map[string]string) map[string]string {
	result := make(map[string]string)

	// Set defaults
	for name, input := range skill.Inputs {
		result[name] = input.Default
	}

	// Override with provided inputs
	for name, value := range inputs {
		result[name] = value
	}

	return result
}

// executeStep executes a single step.
func (i *Invoker) executeStep(step Step, params map[string]string) (StepResult, error) {
	result := StepResult{
		StepName: step.Name,
		Success:  false,
	}

	// Check condition
	if step.If != "" && !i.evaluateCondition(step.If, params) {
		result.Skipped = true
		result.Success = true
		return result, nil
	}

	// Handle different step types
	switch {
	case step.Tool != "":
		// Tool step - would use registry
		result.Output = fmt.Sprintf("Tool: %s", step.Tool)
		result.Success = true

	case step.Command != "":
		// Shell command step
		cmd := i.interpolate(step.Command, params)
		output, err := i.executeShell(cmd)
		result.Output = output
		if err != nil {
			result.Error = err.Error()
			return result, err
		}
		result.Success = true

	default:
		return result, fmt.Errorf("step has no tool or command")
	}

	return result, nil
}

// evaluateCondition evaluates a simple condition string.
func (i *Invoker) evaluateCondition(condition string, params map[string]string) bool {
	// Simple condition evaluation
	// Supports: var == "value", var != "", etc.
	parts := strings.Fields(condition)
	if len(parts) >= 3 {
		left := i.interpolate(parts[0], params)
		op := parts[1]
		right := strings.Trim(i.interpolate(parts[2], params), `"'`)

		switch op {
		case "==", "=":
			return left == right
		case "!=":
			return left != right
		}
	}
	return true
}

// interpolate replaces ${var} placeholders.
func (i *Invoker) interpolate(s string, params map[string]string) string {
	for key, value := range params {
		placeholder := fmt.Sprintf("${%s}", key)
		s = strings.ReplaceAll(s, placeholder, value)
	}
	return s
}

// executeShell runs a shell command.
func (i *Invoker) executeShell(command string) (string, error) {
	cmd := exec.Command("sh", "-c", command)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// Result is the outcome of skill invocation.
type Result struct {
	SkillName   string
	Success     bool
	Error       string
	Outputs     map[string]string
	StepResults []StepResult
}

// StepResult is the outcome of a single step.
type StepResult struct {
	StepName string
	Success  bool
	Skipped  bool
	Output   string
	Error    string
}
