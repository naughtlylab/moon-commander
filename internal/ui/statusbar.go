package ui

import "fmt"

const keyHints = "tab switch · enter open · bksp up · space select · q quit"

// renderStatus draws the one-line status bar spanning the full width.
func (m RootModel) renderStatus() string {
	text := sanitize(m.StatusText)
	if text == "" {
		p := m.pane(m.Active)
		text = fmt.Sprintf("%d items · %d selected  │  %s", len(p.Entries), len(p.Selected), keyHints)
	}
	return m.Styles.Status.Render(fit(" "+text, m.Width))
}
