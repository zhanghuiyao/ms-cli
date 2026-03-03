package skills

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// SkillManifest represents a skill definition file.
type SkillManifest struct {
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Version     string                 `yaml:"version"`
	Author      string                 `yaml:"author"`
	Inputs      map[string]SkillInput  `yaml:"inputs"`
	Outputs     []string               `yaml:"outputs"`
	Steps       []SkillStep            `yaml:"steps"`
}

// SkillInput defines an input parameter.
type SkillInput struct {
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"`
	Default     string `yaml:"default"`
}

// SkillStep represents a step in a skill.
type SkillStep struct {
	Name    string                 `yaml:"name"`
	Tool    string                 `yaml:"tool,omitempty"`
	Command string                 `yaml:"command,omitempty"`
	Script  string                 `yaml:"script,omitempty"`
	Params  map[string]interface{} `yaml:"params,omitempty"`
	If      string                 `yaml:"if,omitempty"`
	Loop    *SkillLoop             `yaml:"loop,omitempty"`
	Timeout int                    `yaml:"timeout,omitempty"`
}

// SkillLoop defines loop configuration.
type SkillLoop struct {
	Over  string `yaml:"over"`
	As    string `yaml:"as"`
	Index string `yaml:"index,omitempty"`
}

// LoadSkillManifest loads a skill from YAML file.
func LoadSkillManifest(path string) (*SkillManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read skill file: %w", err)
	}

	var manifest SkillManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse skill YAML: %w", err)
	}

	// Validate
	if err := manifest.Validate(); err != nil {
		return nil, fmt.Errorf("validate skill: %w", err)
	}

	return &manifest, nil
}

// Validate checks if the skill manifest is valid.
func (m *SkillManifest) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("skill name is required")
	}
	if m.Version == "" {
		return fmt.Errorf("skill version is required")
	}
	if len(m.Steps) == 0 {
		return fmt.Errorf("skill must have at least one step")
	}

	for i, step := range m.Steps {
		if step.Name == "" {
			return fmt.Errorf("step %d: name is required", i)
		}
		if step.Tool == "" && step.Command == "" && step.Script == "" {
			return fmt.Errorf("step %d: must have tool, command, or script", i)
		}
	}

	return nil
}

// ToSkill converts manifest to Skill.
func (m *SkillManifest) ToSkill() *Skill {
	skill := &Skill{
		Name:        m.Name,
		Description: m.Description,
		Version:     m.Version,
		Steps:       make([]Step, len(m.Steps)),
		Inputs:      make(map[string]Input),
		Outputs:     m.Outputs,
	}

	// Convert inputs
	for name, input := range m.Inputs {
		skill.Inputs[name] = Input{
			Type:        input.Type,
			Description: input.Description,
			Required:    input.Required,
			Default:     input.Default,
		}
	}

	// Convert steps
	for i, s := range m.Steps {
		skill.Steps[i] = Step{
			Name:    s.Name,
			Tool:    s.Tool,
			Command: s.Command,
			Params:  stringifyParams(s.Params),
			If:      s.If,
		}

		if s.Loop != nil {
			skill.Steps[i].Loop = &LoopConfig{
				Over:  s.Loop.Over,
				As:    s.Loop.As,
				Index: s.Loop.Index,
			}
		}
	}

	return skill
}

// stringifyParams converts interface{} params to strings.
func stringifyParams(params map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for k, v := range params {
		switch val := v.(type) {
		case string:
			result[k] = val
		default:
			result[k] = fmt.Sprintf("%v", val)
		}
	}
	return result
}

// SkillLoader handles loading skills from directories.
type SkillLoader struct {
	basePath string
}

// NewSkillLoader creates a new skill loader.
func NewSkillLoader(basePath string) *SkillLoader {
	return &SkillLoader{basePath: basePath}
}

// LoadAll loads all skills from the skills directory.
func (l *SkillLoader) LoadAll() (map[string]*SkillManifest, error) {
	skillsDir := filepath.Join(l.basePath, "skills")

	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]*SkillManifest), nil
		}
		return nil, err
	}

	skills := make(map[string]*SkillManifest)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		manifestPath := filepath.Join(skillsDir, entry.Name(), "skill.yaml")
		manifest, err := LoadSkillManifest(manifestPath)
		if err != nil {
			continue // Skip invalid skills
		}

		skills[manifest.Name] = manifest
	}

	return skills, nil
}

// Load loads a single skill by name.
func (l *SkillLoader) Load(name string) (*SkillManifest, error) {
	manifestPath := filepath.Join(l.basePath, "skills", name, "skill.yaml")
	return LoadSkillManifest(manifestPath)
}

// Save saves a skill manifest to file.
func (l *SkillLoader) Save(manifest *SkillManifest) error {
	if err := manifest.Validate(); err != nil {
		return err
	}

	skillsDir := filepath.Join(l.basePath, "skills", manifest.Name)
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return fmt.Errorf("create skill directory: %w", err)
	}

	manifestPath := filepath.Join(skillsDir, "skill.yaml")
	data, err := yaml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("marshal skill: %w", err)
	}

	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("write skill file: %w", err)
	}

	return nil
}

// ExampleSkillYAML is an example skill definition.
const ExampleSkillYAML = `name: hello-world
description: A simple hello world skill
version: "1.0.0"
author: ms-cli

inputs:
  name:
    type: string
    description: Name to greet
    required: false
    default: "World"

outputs:
  - greeting

steps:
  - name: greet
    command: echo "Hello, ${name}!"
`
