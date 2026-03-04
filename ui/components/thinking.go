package components

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var thinkingStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("205")).
	Bold(true)

// ThinkingAnimator renders an animated "Thinking..." indicator.
type ThinkingAnimator struct {
	spinner    spinner.Model
	frames     []string
	frameIndex int
	text       string
	isRunning  bool
}

// ThinkingMsg is sent on each animation tick.
type ThinkingMsg struct {
	Frame string
}

// NewThinkingAnimator creates a new thinking animator with dot spinner.
func NewThinkingAnimator() ThinkingAnimator {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = thinkingStyle

	return ThinkingAnimator{
		spinner: s,
		frames: []string{
			"⣽ Thinking...",
			"⣾ Thinking...",
			"⣷ Thinking...",
			"⣯ Thinking...",
			"⣟ Thinking...",
			"⡿ Thinking...",
			"⢿ Thinking...",
			"⣻ Thinking...",
		},
		frameIndex: 0,
		text:       "Thinking...",
		isRunning:  false,
	}
}

// Start begins the animation.
func (t *ThinkingAnimator) Start() {
	t.isRunning = true
	t.frameIndex = 0
}

// Stop halts the animation.
func (t *ThinkingAnimator) Stop() {
	t.isRunning = false
}

// IsRunning returns true if animation is active.
func (t ThinkingAnimator) IsRunning() bool {
	return t.isRunning
}

// CurrentFrame returns the current animation frame.
func (t ThinkingAnimator) CurrentFrame() string {
	if !t.isRunning {
		return ""
	}
	return t.frames[t.frameIndex]
}

// Update advances the animation.
func (t ThinkingAnimator) Update(msg tea.Msg) (ThinkingAnimator, tea.Cmd) {
	switch msg.(type) {
	case ThinkingMsg:
		if t.isRunning {
			t.frameIndex = (t.frameIndex + 1) % len(t.frames)
			return t, t.tick()
		}
	}

	var cmd tea.Cmd
	t.spinner, cmd = t.spinner.Update(msg)
	return t, cmd
}

// View renders the current frame with spinner.
func (t ThinkingAnimator) View() string {
	if !t.isRunning {
		return ""
	}
	return thinkingStyle.Render(t.frames[t.frameIndex])
}

// tick creates a command that sends a ThinkingMsg after 80ms.
func (t ThinkingAnimator) tick() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(time.Time) tea.Msg {
		return ThinkingMsg{}
	})
}

// Init returns the initial command to start animation.
func (t ThinkingAnimator) Init() tea.Cmd {
	return tea.Batch(
		t.spinner.Tick,
		t.tick(),
	)
}

// WithText creates a new animator with custom text.
func (t ThinkingAnimator) WithText(text string) ThinkingAnimator {
	// Rebuild frames with custom text
	prefixes := []string{"⣽", "⣾", "⣷", "⣯", "⣟", "⡿", "⢿", "⣻"}
	newFrames := make([]string, len(prefixes))
	for i, p := range prefixes {
		newFrames[i] = p + " " + text
	}
	
	return ThinkingAnimator{
		spinner:    t.spinner,
		frames:     newFrames,
		frameIndex: t.frameIndex,
		text:       text,
		isRunning:  t.isRunning,
	}
}
