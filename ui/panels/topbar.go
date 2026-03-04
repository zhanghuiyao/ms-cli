package panels

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/vigo999/ms-cli/ui/model"
)

var (
	topBarStyle = lipgloss.NewStyle().
			Padding(0, 1)

	brandStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

	stepStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)

	stepDoneStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("114"))

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("208")).
			Bold(true)

	statusDoneStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("114"))

	sepStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("238"))

	dividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("238"))

	bannerLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240"))

	bannerValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	bannerDimStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Italic(true)
)

// RenderTopBar renders the top status bar.
func RenderTopBar(s model.State, width int) string {
	sep := sepStyle.Render("│")

	// Build right side with model info and step progress
	rightParts := []string{
		infoStyle.Render("model:"),
		infoStyle.Render(s.Model.Name),
		sep,
		infoStyle.Render(fmt.Sprintf("ctx: %s/%s", formatTokens(s.Model.CtxUsed), formatTokens(s.Model.CtxMax))),
		sep,
		infoStyle.Render(fmt.Sprintf("tokens: %s", formatTokens(s.Model.TokensUsed))),
	}

	// Add status if present
	if s.Model.Status != "" {
		if s.Model.Status == "Done" {
			rightParts = append(rightParts, sep, statusDoneStyle.Render("✓ "+s.Model.Status))
		} else if s.Model.Status == "Thinking..." {
			rightParts = append(rightParts, sep, statusStyle.Render("◐ "+s.Model.Status))
		} else {
			rightParts = append(rightParts, sep, statusStyle.Render(s.Model.Status))
		}
	}

	// Add step progress if active
	if s.StepProgress.IsActive && s.StepProgress.TotalSteps > 0 {
		stepStr := fmt.Sprintf("step: %d/%d", s.StepProgress.CurrentStep, s.StepProgress.TotalSteps)
		if s.StepProgress.CurrentStep == s.StepProgress.TotalSteps {
			rightParts = append(rightParts, sep, stepDoneStyle.Render(stepStr))
		} else {
			rightParts = append(rightParts, sep, stepStyle.Render(stepStr))
		}
	}

	// Line 1: brand + info
	left := brandStyle.Render(s.Version)
	right := strings.Join(rightParts, " ")

	gap := width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}
	pad := lipgloss.NewStyle().Width(gap).Render("")
	line1 := topBarStyle.Render(left + pad + right)

	divider := dividerStyle.Render(repeatChar("━", width))

	// Line 2: workdir + repo
	left2 := bannerLabelStyle.Render("cwd:") + " " + bannerValueStyle.Render(shortenPath(s.WorkDir))
	right2 := bannerDimStyle.Render(s.RepoURL)

	gap2 := width - lipgloss.Width(left2) - lipgloss.Width(right2) - 2
	if gap2 < 1 {
		gap2 = 1
	}
	pad2 := lipgloss.NewStyle().Width(gap2).Render("")
	line2 := topBarStyle.Render(left2 + pad2 + right2)

	return line1 + "\n" + line2 + "\n" + divider
}

func shortenPath(p string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return p
	}
	if strings.HasPrefix(p, home) {
		return "~" + p[len(home):]
	}
	return p
}

func formatTokens(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fk", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

func repeatChar(ch string, n int) string {
	return strings.Repeat(ch, n)
}
