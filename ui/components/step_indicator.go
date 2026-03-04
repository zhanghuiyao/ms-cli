package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	stepActiveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)

	stepDoneStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("114"))

	stepPendingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	stepErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	stepNumberStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250"))

	stepInfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))
)

// StepStatus represents the status of a step.
type StepStatus int

const (
	StepPending StepStatus = iota
	StepRunning
	StepDone
	StepError
)

// Step represents a single step in a process.
type Step struct {
	Number  int
	Name    string
	Status  StepStatus
}

// StepIndicator shows progress through a multi-step process.
type StepIndicator struct {
	steps       []Step
	currentStep int
	showDetails bool
}

// NewStepIndicator creates a new step indicator.
func NewStepIndicator(stepNames []string) StepIndicator {
	steps := make([]Step, len(stepNames))
	for i, name := range stepNames {
		steps[i] = Step{
			Number: i + 1,
			Name:   name,
			Status: StepPending,
		}
	}
	return StepIndicator{
		steps:       steps,
		currentStep: 0,
		showDetails: true,
	}
}

// SetCurrent sets the currently running step (1-indexed).
func (s *StepIndicator) SetCurrent(n int) {
	s.currentStep = n
	for i := range s.steps {
		if i < n-1 {
			s.steps[i].Status = StepDone
		} else if i == n-1 {
			s.steps[i].Status = StepRunning
		} else {
			s.steps[i].Status = StepPending
		}
	}
}

// MarkDone marks the current step as complete and moves to next.
func (s *StepIndicator) MarkDone() {
	if s.currentStep > 0 && s.currentStep <= len(s.steps) {
		s.steps[s.currentStep-1].Status = StepDone
	}
}

// MarkError marks the current step as errored.
func (s *StepIndicator) MarkError() {
	if s.currentStep > 0 && s.currentStep <= len(s.steps) {
		s.steps[s.currentStep-1].Status = StepError
	}
}

// SetShowDetails controls whether to show step names.
func (s *StepIndicator) SetShowDetails(show bool) {
	s.showDetails = show
}

// View renders the step indicator.
func (s StepIndicator) View() string {
	if len(s.steps) == 0 {
		return ""
	}

	var parts []string
	
	// Show compact version in top bar
	if !s.showDetails {
		return s.compactView()
	}

	// Show full step list
	for _, step := range s.steps {
		parts = append(parts, s.renderStep(step))
	}

	return "\n" + lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// compactView shows a compact "Step 3/10" style indicator.
func (s StepIndicator) compactView() string {
	done := 0
	for _, step := range s.steps {
		if step.Status == StepDone {
			done++
		}
	}
	
	status := stepInfoStyle.Render(fmt.Sprintf("Step %d/%d", s.currentStep, len(s.steps)))
	
	// Add progress bar
	progress := s.renderProgressBar()
	
	return lipgloss.JoinHorizontal(lipgloss.Left, status, " ", progress)
}

// renderProgressBar creates a simple text progress bar.
func (s StepIndicator) renderProgressBar() string {
	total := len(s.steps)
	if total == 0 {
		return ""
	}
	
	done := 0
	for _, step := range s.steps {
		if step.Status == StepDone {
			done++
		}
	}
	
	// Simple bar: ▓▓▓░░░
	barWidth := 10
	filled := (done * barWidth) / total
	if filled > barWidth {
		filled = barWidth
	}
	
	bar := ""
	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar += "▓"
		} else {
			bar += "░"
		}
	}
	
	return stepDoneStyle.Render(bar[:filled]) + stepPendingStyle.Render(bar[filled:])
}

// renderStep renders a single step line.
func (s StepIndicator) renderStep(step Step) string {
	var icon, style string
	
	switch step.Status {
	case StepRunning:
		icon = "⣾"
		style = stepActiveStyle.Render(fmt.Sprintf("%s Step %d: %s", icon, step.Number, step.Name))
	case StepDone:
		icon = "✓"
		style = stepDoneStyle.Render(fmt.Sprintf("%s Step %d: %s", icon, step.Number, step.Name))
	case StepError:
		icon = "✗"
		style = stepErrorStyle.Render(fmt.Sprintf("%s Step %d: %s", icon, step.Number, step.Name))
	default:
		icon = "○"
		style = stepPendingStyle.Render(fmt.Sprintf("%s Step %d: %s", icon, step.Number, step.Name))
	}
	
	return "  " + style
}

// CurrentStep returns the current step number.
func (s StepIndicator) CurrentStep() int {
	return s.currentStep
}

// TotalSteps returns the total number of steps.
func (s StepIndicator) TotalSteps() int {
	return len(s.steps)
}

// IsComplete returns true if all steps are done.
func (s StepIndicator) IsComplete() bool {
	for _, step := range s.steps {
		if step.Status != StepDone {
			return false
		}
	}
	return true
}
