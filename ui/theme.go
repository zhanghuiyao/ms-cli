package ui

import (
	"os"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
)

// Theme defines the color scheme for the UI.
type Theme struct {
	Name       string     `yaml:"name"`
	Primary    Color      `yaml:"primary"`
	Secondary  Color      `yaml:"secondary"`
	Success    Color      `yaml:"success"`
	Error      Color      `yaml:"error"`
	Warning    Color      `yaml:"warning"`
	Info       Color      `yaml:"info"`
	Background Color      `yaml:"background"`
	Foreground Color      `yaml:"foreground"`
	Muted      Color      `yaml:"muted"`
	Border     Color      `yaml:"border"`
}

// Color represents a color in the theme.
type Color struct {
	Hex string `yaml:"hex"`
	lipgloss.Color
}

// UnmarshalYAML implements custom YAML unmarshaling.
func (c *Color) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var hex string
	if err := unmarshal(&hex); err != nil {
		return err
	}
	c.Hex = hex
	c.Color = lipgloss.Color(hex)
	return nil
}

// DefaultTheme returns the default theme.
func DefaultTheme() Theme {
	return Theme{
		Name:       "default",
		Primary:    Color{Hex: "#7B68EE", Color: lipgloss.Color("#7B68EE")},
		Secondary:  Color{Hex: "#00CED1", Color: lipgloss.Color("#00CED1")},
		Success:    Color{Hex: "#32CD32", Color: lipgloss.Color("#32CD32")},
		Error:      Color{Hex: "#DC143C", Color: lipgloss.Color("#DC143C")},
		Warning:    Color{Hex: "#FFD700", Color: lipgloss.Color("#FFD700")},
		Info:       Color{Hex: "#1E90FF", Color: lipgloss.Color("#1E90FF")},
		Background: Color{Hex: "#000000", Color: lipgloss.Color("#000000")},
		Foreground: Color{Hex: "#FFFFFF", Color: lipgloss.Color("#FFFFFF")},
		Muted:      Color{Hex: "#808080", Color: lipgloss.Color("#808080")},
		Border:     Color{Hex: "#444444", Color: lipgloss.Color("#444444")},
	}
}

// DarkTheme returns a dark theme variant.
func DarkTheme() Theme {
	t := DefaultTheme()
	t.Name = "dark"
	return t
}

// LightTheme returns a light theme.
func LightTheme() Theme {
	return Theme{
		Name:       "light",
		Primary:    Color{Hex: "#5B4FC4", Color: lipgloss.Color("#5B4FC4")},
		Secondary:  Color{Hex: "#008B8B", Color: lipgloss.Color("#008B8B")},
		Success:    Color{Hex: "#228B22", Color: lipgloss.Color("#228B22")},
		Error:      Color{Hex: "#B22222", Color: lipgloss.Color("#B22222")},
		Warning:    Color{Hex: "#DAA520", Color: lipgloss.Color("#DAA520")},
		Info:       Color{Hex: "#4169E1", Color: lipgloss.Color("#4169E1")},
		Background: Color{Hex: "#FFFFFF", Color: lipgloss.Color("#FFFFFF")},
		Foreground: Color{Hex: "#000000", Color: lipgloss.Color("#000000")},
		Muted:      Color{Hex: "#696969", Color: lipgloss.Color("#696969")},
		Border:     Color{Hex: "#CCCCCC", Color: lipgloss.Color("#CCCCCC")},
	}
}

// LoadTheme loads a theme from a YAML file.
func LoadTheme(path string) (Theme, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Theme{}, err
	}
	
	var theme Theme
	if err := yaml.Unmarshal(data, &theme); err != nil {
		return Theme{}, err
	}
	
	return theme, nil
}

// Save saves the theme to a YAML file.
func (t Theme) Save(path string) error {
	data, err := yaml.Marshal(t)
	if err != nil {
		return err
	}
	
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	
	return os.WriteFile(path, data, 0644)
}

// GetBuiltinThemes returns all built-in themes.
func GetBuiltinThemes() []Theme {
	return []Theme{
		DefaultTheme(),
		DarkTheme(),
		LightTheme(),
	}
}
