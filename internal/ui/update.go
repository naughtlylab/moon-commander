package ui

import (
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

// update is the pure state transition for RootModel. It performs no I/O;
// disk access is returned as tea.Cmd values. After every transition the
// scroll windows are re-synced (SPEC §3.6).
func (m RootModel) update(msg tea.Msg) (RootModel, tea.Cmd) {
	m, cmd := m.step(msg)
	rows := visibleRows(m.Height)
	m.Left.syncScroll(rows)
	m.Right.syncScroll(rows)
	return m, cmd
}

// visibleRows is the number of entry rows a pane can show (SPEC §3.6).
func visibleRows(height int) int {
	return max(1, height-4)
}

// syncScroll keeps the cursor inside the visible window and the window
// inside the entry list.
func (p *PaneModel) syncScroll(rows int) {
	if p.Cursor < p.ScrollOffset {
		p.ScrollOffset = p.Cursor
	}
	if p.Cursor >= p.ScrollOffset+rows {
		p.ScrollOffset = p.Cursor - rows + 1
	}
	p.ScrollOffset = min(p.ScrollOffset, max(0, len(p.Entries)-rows))
	p.ScrollOffset = max(0, p.ScrollOffset)
}

func (m RootModel) step(msg tea.Msg) (RootModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.WindowSizeMsg:
		m.Width, m.Height = msg.Width, msg.Height
		return m, nil
	case DirLoadedMsg:
		p := m.pane(msg.Pane)
		p.Cwd = msg.Path
		p.Entries = msg.Entries
		p.Cursor = 0
		p.ScrollOffset = 0
		p.Loading = false
		p.Err = nil
		return m, nil
	case DirLoadErrMsg:
		p := m.pane(msg.Pane)
		p.Loading = false
		p.Err = msg.Err
		p.clampCursor()
		m.StatusText = msg.Err.Error()
		return m, nil
	}
	return m, nil
}

func (m RootModel) handleKey(msg tea.KeyMsg) (RootModel, tea.Cmd) {
	p := m.pane(m.Active)
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "tab":
		if m.Active == LeftPane {
			m.Active = RightPane
		} else {
			m.Active = LeftPane
		}
	case "up", "k":
		p.Cursor--
		p.clampCursor()
	case "down", "j":
		p.Cursor++
		p.clampCursor()
	case " ", "space":
		p.toggleSelected()
		p.Cursor++
		p.clampCursor()
	case "enter", "right", "l":
		if p.Loading || len(p.Entries) == 0 {
			return m, nil
		}
		e := p.Entries[p.Cursor]
		if !e.IsDir {
			return m, nil
		}
		p.Loading = true
		return m, loadDirCmd(m.FS, p.ID, e.Path)
	case "backspace", "left", "h":
		if p.Loading {
			return m, nil
		}
		p.Loading = true
		return m, loadDirCmd(m.FS, p.ID, filepath.Dir(p.Cwd))
	}
	return m, nil
}

// toggleSelected flips the selection of the entry under the cursor. The map
// is copied first so previous RootModel values are never mutated.
func (p *PaneModel) toggleSelected() {
	if len(p.Entries) == 0 {
		return
	}
	sel := make(map[string]struct{}, len(p.Selected)+1)
	for k := range p.Selected {
		sel[k] = struct{}{}
	}
	path := p.Entries[p.Cursor].Path
	if _, ok := sel[path]; ok {
		delete(sel, path)
	} else {
		sel[path] = struct{}{}
	}
	p.Selected = sel
}
