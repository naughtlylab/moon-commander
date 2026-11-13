package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

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

	// Tab row blocks (SPEC §6 Tabs).
	TabInactive     lipgloss.Style // any non-active tab
	TabActive       lipgloss.Style // active tab of the active pane
	TabActiveMuted  lipgloss.Style // active tab of the inactive pane
	MarkActiveTabNC bool           // no colour available: bracket the active tab label
}

// DefaultStyles builds the styles for the default (stdout) renderer.
func DefaultStyles() Styles {
	return NewStyles(lipgloss.DefaultRenderer())
}

// NewStyles builds the styles for a specific renderer (useful for tests that
// need a fixed color profile).
func NewStyles(r *lipgloss.Renderer) Styles {
	var (
		accent      = lipgloss.Color("39")
		mutedAccent = lipgloss.Color("24")
		dim         = lipgloss.Color("240")
		disabledBg  = lipgloss.Color("238")
		lightText   = lipgloss.Color("252")
		darkText    = lipgloss.Color("16")
	)
	border := r.NewStyle().Border(lipgloss.RoundedBorder())
	return Styles{
		BorderActive:   border.BorderForeground(accent),
		BorderInactive: border.BorderForeground(dim),
		Header:         r.NewStyle().Bold(true),
		CursorActive:   r.NewStyle().Reverse(true),
		CursorInactive: r.NewStyle().Underline(true),
		Error:          r.NewStyle().Foreground(lipgloss.Color("203")),
		Status:         r.NewStyle().Background(lipgloss.Color("236")).Foreground(lightText),

		TabInactive:     r.NewStyle().Background(disabledBg).Foreground(lightText),
		TabActive:       r.NewStyle().Background(accent).Foreground(darkText).Bold(true),
		TabActiveMuted:  r.NewStyle().Background(mutedAccent).Foreground(lightText).Bold(true),
		MarkActiveTabNC: r.ColorProfile() == termenv.Ascii,
	}
}
