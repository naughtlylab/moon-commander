package ui

import "fmt"

const keyHints = "tab switch · enter open · bksp up · space select · ^t new tab · ] [ tabs · q quit"

// renderStatus draws the one-line status bar spanning the full width.
func (m RootModel) renderStatus() string {
	text := sanitize(m.StatusText)
	if text == "" {
		p := m.pane(m.Active)
		t := p.tab()
		text = fmt.Sprintf("%d items · %d selected  │  %s", len(t.Entries), len(t.Selected), keyHints)
		if len(p.Tabs) > 1 {
			text = fmt.Sprintf("tab %d/%d · ", p.ActiveTab+1, len(p.Tabs)) + text
		}
	}
	return m.Styles.Status.Render(fit(" "+text, m.Width))
}
