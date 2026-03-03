package ui

import (
	"testing"
)

func TestDefaultTheme(t *testing.T) {
	theme := DefaultTheme()

	if theme.Name != "default" {
		t.Errorf("expected name 'default', got '%s'", theme.Name)
	}

	// Verify all colors are set
	if theme.Primary == "" {
		t.Error("primary color should be set")
	}

	if theme.Success == "" {
		t.Error("success color should be set")
	}

	if theme.Error == "" {
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

func TestHighContrastTheme(t *testing.T) {
	theme := HighContrastTheme()

	if theme.Name != "high-contrast" {
		t.Errorf("expected name 'high-contrast', got '%s'", theme.Name)
	}
}

func TestGetTheme(t *testing.T) {
	// Test existing theme
	theme := GetTheme("dark")
	if theme.Name != "dark" {
		t.Errorf("expected dark theme, got '%s'", theme.Name)
	}

	// Test non-existent theme returns default
	theme = GetTheme("nonexistent")
	if theme.Name != "default" {
		t.Errorf("expected default theme for unknown name, got '%s'", theme.Name)
	}
}

func TestListThemes(t *testing.T) {
	themes := ListThemes()

	expectedThemes := []string{"default", "dark", "light", "high-contrast"}

	if len(themes) != len(expectedThemes) {
		t.Errorf("expected %d themes, got %d", len(expectedThemes), len(themes))
	}

	// Verify all expected themes exist
	for _, expected := range expectedThemes {
		found := false
		for _, actual := range themes {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected theme '%s' not found", expected)
		}
	}
}
