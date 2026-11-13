package ui

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// View renders both panes side by side above a one-line status bar
// (SPEC §6). It is a pure function of the model.
func (m RootModel) View() string {
	if m.Width == 0 {
		return "Initializing…"
	}
	lw := m.Width / 2
	panes := lipgloss.JoinHorizontal(lipgloss.Top,
		m.renderPane(m.Left, lw),
		m.renderPane(m.Right, m.Width-lw),
	)
	return panes + "\n" + m.renderStatus()
}

// fit truncates s to width w (with "…") and right-pads it to exactly w cells.
func fit(s string, w int) string {
	if w <= 0 {
		return ""
	}
	s = ansi.Truncate(s, w, "…")
	return s + strings.Repeat(" ", max(0, w-ansi.StringWidth(s)))
}

// fitLeft is like fit but drops leading characters, keeping the tail visible.
func fitLeft(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if ansi.StringWidth(s) <= w {
		return fit(s, w)
	}
	r := []rune(s)
	for len(r) > 0 && ansi.StringWidth(string(r)) > w-1 {
		r = r[1:]
	}
	return fit("…"+string(r), w)
}

// sanitize replaces control characters (tabs, newlines, escapes) that would
// corrupt the layout if printed verbatim.
func sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return '?'
		}
		return r
	}, s)
}
