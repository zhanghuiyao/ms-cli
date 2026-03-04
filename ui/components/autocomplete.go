package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// AutoComplete provides command completion for the input.
type AutoComplete struct {
	input       textinput.Model
	suggestions []string
	selected    int
	showing     bool
	commands    []string
}

// NewAutoComplete creates a new auto-complete component.
func NewAutoComplete() AutoComplete {
	ac := AutoComplete{
		input:    textinput.New(),
		commands: defaultCommands(),
	}
	ac.input.Placeholder = "Type / for commands..."
	ac.input.Prompt = "> "
	ac.input.Focus()
	ac.input.CharLimit = 2000
	return ac
}

func defaultCommands() []string {
	return []string{
		"/model",
		"/model list",
		"/model use",
		"/compact",
		"/clear",
		"/new",
		"/help",
		"/save",
		"/load",
		"/history",
		"/search",
		"/tools",
		"/temperature",
		"/tokens",
		"/theme",
		"/roadmap",
		"/roadmap status",
		"/weekly",
		"/weekly status",
	}
}

// Update handles input updates and completion logic.
func (ac AutoComplete) Update(msg tea.Msg) (AutoComplete, tea.Cmd) {
	var cmd tea.Cmd
	ac.input, cmd = ac.input.Update(msg)

	// Update suggestions based on input
	val := ac.input.Value()
	if strings.HasPrefix(val, "/") {
		ac.suggestions = ac.filter(val)
		ac.showing = len(ac.suggestions) > 0
		if ac.selected >= len(ac.suggestions) {
			ac.selected = 0
		}
	} else {
		ac.showing = false
	}

	return ac, cmd
}

func (ac AutoComplete) filter(input string) []string {
	var matches []string
	input = strings.ToLower(input)
	for _, cmd := range ac.commands {
		if strings.HasPrefix(strings.ToLower(cmd), input) {
			matches = append(matches, cmd)
		}
	}
	return matches
}

// NextSuggestion cycles to the next suggestion.
func (ac AutoComplete) NextSuggestion() AutoComplete {
	if len(ac.suggestions) > 0 {
		ac.selected = (ac.selected + 1) % len(ac.suggestions)
	}
	return ac
}

// PrevSuggestion cycles to the previous suggestion.
func (ac AutoComplete) PrevSuggestion() AutoComplete {
	if len(ac.suggestions) > 0 {
		ac.selected--
		if ac.selected < 0 {
			ac.selected = len(ac.suggestions) - 1
		}
	}
	return ac
}

// AcceptSuggestion applies the current suggestion.
func (ac AutoComplete) AcceptSuggestion() AutoComplete {
	if ac.showing && len(ac.suggestions) > 0 {
		ac.input.SetValue(ac.suggestions[ac.selected] + " ")
		ac.showing = false
		ac.selected = 0
	}
	return ac
}

// View renders the auto-complete component.
func (ac AutoComplete) View() string {
	view := ac.input.View()

	if ac.showing {
		view += "\n"
		for i, sug := range ac.suggestions {
			if i > 5 {
				view += "  ..."
				break
			}
			if i == ac.selected {
				view += "▸ " + sug + "\n"
			} else {
				view += "  " + sug + "\n"
			}
		}
	}

	return view
}

// Showing returns true if suggestions are being shown.
func (ac AutoComplete) Showing() bool {
	return ac.showing
}

// Value returns the input value.
func (ac AutoComplete) Value() string {
	return ac.input.Value()
}

// SetValue sets the input value.
func (ac AutoComplete) SetValue(s string) AutoComplete {
	ac.input.SetValue(s)
	return ac
}

// Focus focuses the input.
func (ac AutoComplete) Focus() tea.Cmd {
	return ac.input.Focus()
}

// Blur removes focus.
func (ac AutoComplete) Blur() {
	ac.input.Blur()
}

// Reset clears the input.
func (ac AutoComplete) Reset() AutoComplete {
	ac.input.Reset()
	ac.showing = false
	ac.selected = 0
	return ac
}
