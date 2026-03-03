package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSkillManifestValidation(t *testing.T) {
	tests := []struct {
		name    string
		manifest *SkillManifest
		wantErr bool
	}{
		{
			name: "valid manifest",
			manifest: &SkillManifest{
				Name:    "test-skill",
				Version: "1.0.0",
				Steps:   []SkillStep{{Name: "step1", Command: "echo test"}},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			manifest: &SkillManifest{
				Version: "1.0.0",
				Steps:   []SkillStep{{Name: "step1", Command: "echo test"}},
			},
			wantErr: true,
		},
		{
			name: "missing version",
			manifest: &SkillManifest{
				Name:  "test-skill",
				Steps: []SkillStep{{Name: "step1", Command: "echo test"}},
			},
			wantErr: true,
		},
		{
			name: "no steps",
			manifest: &SkillManifest{
				Name:    "test-skill",
				Version: "1.0.0",
			},
			wantErr: true,
		},
		{
			name: "step missing name",
			manifest: &SkillManifest{
				Name:    "test-skill",
				Version: "1.0.0",
				Steps:   []SkillStep{{Command: "echo test"}},
			},
			wantErr: true,
		},
		{
			name: "step missing action",
			manifest: &SkillManifest{
				Name:    "test-skill",
				Version: "1.0.0",
				Steps:   []SkillStep{{Name: "step1"}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.manifest.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadSkillManifest(t *testing.T) {
	// Create temp skill file
	tmpDir := t.TempDir()
	skillContent := `name: test-skill
description: A test skill
version: "1.0.0"
author: test

inputs:
  name:
    type: string
    description: Name to greet
    required: false
    default: "World"

steps:
  - name: greet
    command: echo "Hello, ${name}!"
`

	skillPath := filepath.Join(tmpDir, "skill.yaml")
	if err := os.WriteFile(skillPath, []byte(skillContent), 0644); err != nil {
		t.Fatal(err)
	}

	manifest, err := LoadSkillManifest(skillPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if manifest.Name != "test-skill" {
		t.Errorf("expected name 'test-skill', got '%s'", manifest.Name)
	}

	if manifest.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", manifest.Version)
	}

	if len(manifest.Steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(manifest.Steps))
	}

	if manifest.Steps[0].Name != "greet" {
		t.Errorf("expected step name 'greet', got '%s'", manifest.Steps[0].Name)
	}
}

func TestLoadSkillManifestNotFound(t *testing.T) {
	_, err := LoadSkillManifest("/nonexistent/path/skill.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestSkillManifestToSkill(t *testing.T) {
	manifest := &SkillManifest{
		Name:        "test-skill",
		Description: "A test skill",
		Version:     "1.0.0",
		Inputs: map[string]SkillInput{
			"name": {
				Type:        "string",
				Description: "Name to greet",
				Required:    false,
				Default:     "World",
			},
		},
		Steps: []SkillStep{
			{
				Name:    "greet",
				Command: "echo Hello",
				Params:  map[string]interface{}{"arg": "value"},
			},
		},
	}

	skill := manifest.ToSkill()

	if skill.Name != "test-skill" {
		t.Errorf("expected name 'test-skill', got '%s'", skill.Name)
	}

	if len(skill.Steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(skill.Steps))
	}

	if len(skill.Inputs) != 1 {
		t.Errorf("expected 1 input, got %d", len(skill.Inputs))
	}
}

func TestSkillLoader(t *testing.T) {
	tmpDir := t.TempDir()
	loader := NewSkillLoader(tmpDir)

	// Create skill directory structure
	skillsDir := filepath.Join(tmpDir, "skills", "test-skill")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		t.Fatal(err)
	}

	skillContent := `name: test-skill
description: A test skill
version: "1.0.0"

steps:
  - name: test
    command: echo test
`

	skillPath := filepath.Join(skillsDir, "skill.yaml")
	if err := os.WriteFile(skillPath, []byte(skillContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Test LoadAll
	skills, err := loader.LoadAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(skills) != 1 {
		t.Errorf("expected 1 skill, got %d", len(skills))
	}

	// Test Load
	manifest, err := loader.Load("test-skill")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if manifest.Name != "test-skill" {
		t.Errorf("expected name 'test-skill', got '%s'", manifest.Name)
	}
}

func TestSkillLoaderSave(t *testing.T) {
	tmpDir := t.TempDir()
	loader := NewSkillLoader(tmpDir)

	manifest := &SkillManifest{
		Name:    "new-skill",
		Version: "1.0.0",
		Steps: []SkillStep{
			{Name: "step1", Command: "echo test"},
		},
	}

	if err := loader.Save(manifest); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file was created
	skillPath := filepath.Join(tmpDir, "skills", "new-skill", "skill.yaml")
	if _, err := os.Stat(skillPath); err != nil {
		t.Errorf("skill file was not created: %v", err)
	}
}

func TestStringifyParams(t *testing.T) {
	params := map[string]interface{}{
		"string": "value",
		"int":    42,
		"bool":   true,
		"float":  3.14,
	}

	result := stringifyParams(params)

	if result["string"] != "value" {
		t.Errorf("expected 'value', got '%s'", result["string"])
	}

	if result["int"] != "42" {
		t.Errorf("expected '42', got '%s'", result["int"])
	}

	if result["bool"] != "true" {
		t.Errorf("expected 'true', got '%s'", result["bool"])
	}
}
