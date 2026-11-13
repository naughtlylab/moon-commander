package ui

import (
	"fmt"
	"strings"

	"github.com/naughtlylab/moon-commander/internal/fs"
)

// sizeColWidth fits "<DIR>" and every humanSize result ("1023K", "1023B").
const sizeColWidth = 5

// renderPane draws a pane's tab row above its bordered box (showing the
// active tab), with an outer width of outerW cells.
func (m RootModel) renderPane(p PaneModel, outerW int) string {
	innerW := max(0, outerW-2)
	rows := visibleRows(m.Height)
	active := p.ID == m.Active
	t := p.tab()

	lines := make([]string, 0, rows+1)
	lines = append(lines, m.Styles.Header.Render(fitLeft(sanitize(t.Cwd), innerW)))

	switch {
	case t.Loading:
		lines = append(lines, fit("Loading…", innerW))
	case t.Err != nil:
		lines = append(lines, m.Styles.Error.Render(fit(sanitize(t.Err.Error()), innerW)))
	default:
		end := min(len(t.Entries), t.ScrollOffset+rows)
		for i := t.ScrollOffset; i < end; i++ {
			e := t.Entries[i]
			_, sel := t.Selected[e.Path]
			row := entryRow(e, sel, innerW)
			if i == t.Cursor {
				if active {
					row = m.Styles.CursorActive.Render(row)
				} else {
					row = m.Styles.CursorInactive.Render(row)
				}
			}
			lines = append(lines, row)
		}
	}
	blank := strings.Repeat(" ", innerW)
	for len(lines) < rows+1 {
		lines = append(lines, blank)
	}

	border := m.Styles.BorderInactive
	if active {
		border = m.Styles.BorderActive
	}
	return m.tabRow(p, outerW, active) + "\n" + border.Render(strings.Join(lines, "\n"))
}

// entryRow formats a single entry as exactly w cells:
// marker, space, name, space, right-aligned size.
func entryRow(e fs.Entry, selected bool, w int) string {
	marker := " "
	if selected {
		marker = "*"
	}
	name := sanitize(e.Name)
	size := humanSize(e.Size)
	if e.IsDir {
		name += "/"
		size = "<DIR>"
	}
	nameW := w - 2 - 1 - sizeColWidth
	if nameW < 4 {
		return fit(marker+" "+name, w)
	}
	return marker + " " + fit(name, nameW) + " " + fmt.Sprintf("%*s", sizeColWidth, size)
}

// humanSize formats n bytes using binary units, at most 5 cells wide.
func humanSize(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%dB", n)
	}
	v := float64(n)
	for _, unit := range "KMGTPE" {
		v /= 1024
		if v < 1024 || unit == 'E' {
			if v < 10 {
				return fmt.Sprintf("%.1f%c", v, unit)
			}
			return fmt.Sprintf("%.0f%c", v, unit)
		}
	}
	return "" // unreachable
}
