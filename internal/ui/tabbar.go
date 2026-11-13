package ui

import (
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const (
	maxTabLabelWidth = 16
	tabGap           = " "
	tabClipLeft      = "… "
	tabClipRight     = " …"
)

// tabSegment is one piece of the tab row; style is nil for plain text.
type tabSegment struct {
	text  string
	style *lipgloss.Style
}

// tabRow renders the one-line row of tab blocks shown above a pane,
// exactly width cells wide (SPEC §6 Tabs).
func (m RootModel) tabRow(p PaneModel, width int, paneActive bool) string {
	if width <= 0 {
		return ""
	}
	var b strings.Builder
	used := 0
	for _, s := range m.tabSegments(p, width, paneActive) {
		used += ansi.StringWidth(s.text)
		if s.style != nil {
			b.WriteString(s.style.Render(s.text))
		} else {
			b.WriteString(s.text)
		}
	}
	b.WriteString(strings.Repeat(" ", max(0, width-used)))
	return b.String()
}

// tabLabel is the display name of a tab: the base of its directory.
func tabLabel(t TabModel) string {
	return ansi.Truncate(sanitize(filepath.Base(t.Cwd)), maxTabLabelWidth, "…")
}

// tabSegments picks the largest window of whole tab blocks around the active
// tab that fits in avail cells, with a plain "…" at each clipped edge.
func (m RootModel) tabSegments(p PaneModel, avail int, paneActive bool) []tabSegment {
	n, a := len(p.Tabs), p.ActiveTab
	blocks := make([]string, n)
	for i, t := range p.Tabs {
		label := tabLabel(t)
		if i == a && m.Styles.MarkActiveTabNC {
			label = "[" + label + "]"
		}
		blocks[i] = " " + label + " "
	}
	activeStyle := m.Styles.TabActiveMuted
	if paneActive {
		activeStyle = m.Styles.TabActive
	}
	styleOf := func(i int) *lipgloss.Style {
		if i == a {
			return &activeStyle
		}
		return &m.Styles.TabInactive
	}

	width := func(lo, hi int) int {
		w := len(tabGap) * (hi - lo)
		for i := lo; i <= hi; i++ {
			w += ansi.StringWidth(blocks[i])
		}
		if lo > 0 {
			w += ansi.StringWidth(tabClipLeft)
		}
		if hi < n-1 {
			w += ansi.StringWidth(tabClipRight)
		}
		return w
	}

	if width(a, a) > avail {
		// Not even the active block fits with its clip markers: show it alone.
		return []tabSegment{{text: ansi.Truncate(blocks[a], avail, "…"), style: styleOf(a)}}
	}
	lo, hi := a, a
	for grown := true; grown; {
		grown = false
		if hi < n-1 && width(lo, hi+1) <= avail {
			hi++
			grown = true
		}
		if lo > 0 && width(lo-1, hi) <= avail {
			lo--
			grown = true
		}
	}

	var segs []tabSegment
	if lo > 0 {
		segs = append(segs, tabSegment{text: tabClipLeft})
	}
	for i := lo; i <= hi; i++ {
		if i > lo {
			segs = append(segs, tabSegment{text: tabGap})
		}
		segs = append(segs, tabSegment{text: blocks[i], style: styleOf(i)})
	}
	if hi < n-1 {
		segs = append(segs, tabSegment{text: tabClipRight})
	}
	return segs
}
