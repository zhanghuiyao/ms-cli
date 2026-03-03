package ui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultTheme(t *testing.T) {
	theme := DefaultTheme()

	if theme.Name != "default" {
		t.Errorf("expected name 'default', got '%s'", theme.Name)
	}

	// Verify all colors are set
	if theme.Primary.Hex == "" {
		t.Error("primary color should be set")
	}

	if theme.Success.Hex == "" {
		t.Error("success color should be set")
	}

	if theme.Error.Hex == "" {
		t.Error("error color should be set")
	}
}

func TestDarkTheme(t *testing.T) {
	theme := DarkTheme()

	if theme.Name != "dark" {
		t.Errorf("expected name 'dark', got '%s'", theme.Name)
	}
}

func TestLightTheme(t *testing.T) {
	theme := LightTheme()

	if theme.Name != "light" {
		t.Errorf("expected name 'light', got '%s'", theme.Name)
	}
}

func TestLoadTheme(t *testing.T) {
	// Skip this test if no theme file exists
	tmpDir := t.TempDir()
	themePath := filepath.Join(tmpDir, "test_theme.yaml")

	// Create a test theme file
	themeContent := `name: test
title: "Test Theme"
primary: "#FF0000"
secondary: "#00FF00"
success: "#0000FF"
error: "#FF00FF"
`
	if err := os.WriteFile(themePath, []byte(themeContent), 0644); err != nil {
		t.Fatalf("failed to create test theme file: %v", err)
	}

	theme, err := LoadTheme(themePath)
	if err != nil {
		t.Errorf("unexpected error loading theme: %v", err)
	}

	if theme.Name != "test" {
		t.Errorf("expected name 'test', got '%s'", theme.Name)
	}
}
