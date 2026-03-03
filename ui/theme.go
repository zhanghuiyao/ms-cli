package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Theme defines the color scheme for the UI.
type Theme struct {
	Name string

	// Primary colors
	Primary   lipgloss.Color
	Secondary lipgloss.Color

	// Background colors
	Background lipgloss.Color
	Surface    lipgloss.Color

	// Text colors
	TextPrimary   lipgloss.Color
	TextSecondary lipgloss.Color
	TextMuted     lipgloss.Color

	// Status colors
	Success lipgloss.Color
	Warning lipgloss.Color
	Error   lipgloss.Color
	Info    lipgloss.Color

	// Accent colors
	Accent1 lipgloss.Color
	Accent2 lipgloss.Color
	Accent3 lipgloss.Color
}

// Themes is a collection of available themes.
var Themes = map[string]Theme{
	"default": DefaultTheme(),
	"dark":    DarkTheme(),
	"light":   LightTheme(),
	"high-contrast": HighContrastTheme(),
}

// DefaultTheme returns the default dark theme.
func DefaultTheme() Theme {
	return Theme{
		Name:          "default",
		Primary:       lipgloss.Color("86"),   // Cyan
		Secondary:     lipgloss.Color("244"),  // Gray
		Background:    lipgloss.Color("0"),    // Black
		Surface:       lipgloss.Color("235"),  // Dark gray
		TextPrimary:   lipgloss.Color("252"),  // White
		TextSecondary: lipgloss.Color("250"),  // Light gray
		TextMuted:     lipgloss.Color("240"),  // Muted gray
		Success:       lipgloss.Color("114"),  // Green
		Warning:       lipgloss.Color("214"),  // Orange
		Error:         lipgloss.Color("196"),  // Red
		Info:          lipgloss.Color("39"),   // Blue
		Accent1:       lipgloss.Color("205"),  // Pink
		Accent2:       lipgloss.Color("39"),   // Blue
		Accent3:       lipgloss.Color("214"),  // Orange
	}
}

// DarkTheme returns a darker variant.
func DarkTheme() Theme {
	return Theme{
		Name:          "dark",
		Primary:       lipgloss.Color("81"),   // Light blue
		Secondary:     lipgloss.Color("245"),  // Gray
		Background:    lipgloss.Color("232"),  // Almost black
		Surface:       lipgloss.Color("236"),  // Dark gray
		TextPrimary:   lipgloss.Color("255"),  // White
		TextSecondary: lipgloss.Color("251"),  // Light gray
		TextMuted:     lipgloss.Color("241"),  // Muted gray
		Success:       lipgloss.Color("82"),   // Bright green
		Warning:       lipgloss.Color("220"),  // Yellow
		Error:         lipgloss.Color("203"),  // Light red
		Info:          lipgloss.Color("33"),   // Cyan
		Accent1:       lipgloss.Color("183"),  // Light purple
		Accent2:       lipgloss.Color("33"),   // Cyan
		Accent3:       lipgloss.Color("215"),  // Light orange
	}
}

// LightTheme returns a light theme.
func LightTheme() Theme {
	return Theme{
		Name:          "light",
		Primary:       lipgloss.Color("25"),   // Blue
		Secondary:     lipgloss.Color("240"),  // Gray
		Background:    lipgloss.Color("255"),  // White
		Surface:       lipgloss.Color("253"),  // Light gray
		TextPrimary:   lipgloss.Color("16"),   // Black
		TextSecondary: lipgloss.Color("238"),  // Dark gray
		TextMuted:     lipgloss.Color("245"),  // Gray
		Success:       lipgloss.Color("28"),   // Green
		Warning:       lipgloss.Color("166"),  // Orange
		Error:         lipgloss.Color("160"),  // Red
		Info:          lipgloss.Color("25"),   // Blue
		Accent1:       lipgloss.Color("162"),  // Purple
		Accent2:       lipgloss.Color("31"),   // Teal
		Accent3:       lipgloss.Color("130"),  // Brown
	}
}

// HighContrastTheme returns a high contrast theme for accessibility.
func HighContrastTheme() Theme {
	return Theme{
		Name:          "high-contrast",
		Primary:       lipgloss.Color("51"),   // Bright cyan
		Secondary:     lipgloss.Color("250"),  // White
		Background:    lipgloss.Color("0"),    // Black
		Surface:       lipgloss.Color("0"),    // Black
		TextPrimary:   lipgloss.Color("15"),   // White
		TextSecondary: lipgloss.Color("250"),  // Light gray
		TextMuted:     lipgloss.Color("248"),  // Gray
		Success:       lipgloss.Color("46"),   // Bright green
		Warning:       lipgloss.Color("226"),  // Bright yellow
		Error:         lipgloss.Color("196"),  // Bright red
		Info:          lipgloss.Color("51"),   // Bright cyan
		Accent1:       lipgloss.Color("213"),  // Bright pink
		Accent2:       lipgloss.Color("51"),   // Bright cyan
		Accent3:       lipgloss.Color("220"),  // Bright yellow
	}
}

// GetTheme returns a theme by name.
func GetTheme(name string) Theme {
	if theme, ok := Themes[name]; ok {
		return theme
	}
	return DefaultTheme()
}

// ListThemes returns available theme names.
func ListThemes() []string {
	names := make([]string, 0, len(Themes))
	for name := range Themes {
		names = append(names, name)
	}
	return names
}
