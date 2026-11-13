package ui

import "github.com/charmbracelet/lipgloss"

// Styles holds every Lip Gloss style used by View. It lives in RootModel so
// there is no package-level style state.
type Styles struct {
	BorderActive   lipgloss.Style
	BorderInactive lipgloss.Style
	Header         lipgloss.Style
	CursorActive   lipgloss.Style
	CursorInactive lipgloss.Style
	Error          lipgloss.Style
	Status         lipgloss.Style
}

// DefaultStyles builds the styles for the default (stdout) renderer.
func DefaultStyles() Styles {
	return NewStyles(lipgloss.DefaultRenderer())
}

// NewStyles builds the styles for a specific renderer (useful for tests that
// need a fixed color profile).
func NewStyles(r *lipgloss.Renderer) Styles {
	border := r.NewStyle().Border(lipgloss.RoundedBorder())
	return Styles{
		BorderActive:   border.BorderForeground(lipgloss.Color("39")),
		BorderInactive: border.BorderForeground(lipgloss.Color("240")),
		Header:         r.NewStyle().Bold(true),
		CursorActive:   r.NewStyle().Reverse(true),
		CursorInactive: r.NewStyle().Underline(true),
		Error:          r.NewStyle().Foreground(lipgloss.Color("203")),
		Status:         r.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("252")),
	}
}
