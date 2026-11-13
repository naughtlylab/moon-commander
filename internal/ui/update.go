package ui

import (
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

// update is the pure state transition for RootModel. It performs no I/O;
// disk access is returned as tea.Cmd values. Tab slices are cloned first so
// the receiver's state is never mutated, and after every transition the
// scroll windows of all tabs are re-synced (SPEC §3.6).
func (m RootModel) update(msg tea.Msg) (RootModel, tea.Cmd) {
	m.Left.cloneTabs()
	m.Right.cloneTabs()
	m, cmd := m.step(msg)
	rows := visibleRows(m.Height)
	for _, p := range []*PaneModel{&m.Left, &m.Right} {
		for i := range p.Tabs {
			p.Tabs[i].syncScroll(rows)
		}
	}
	return m, cmd
}

// visibleRows is the number of entry rows a pane can show (SPEC §3.6).
func visibleRows(height int) int {
	return max(1, height-5)
}

// syncScroll keeps the cursor inside the visible window and the window
// inside the entry list.
func (t *TabModel) syncScroll(rows int) {
	if t.Cursor < t.ScrollOffset {
		t.ScrollOffset = t.Cursor
	}
	if t.Cursor >= t.ScrollOffset+rows {
		t.ScrollOffset = t.Cursor - rows + 1
	}
	t.ScrollOffset = min(t.ScrollOffset, max(0, len(t.Entries)-rows))
	t.ScrollOffset = max(0, t.ScrollOffset)
}

func (m RootModel) step(msg tea.Msg) (RootModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.WindowSizeMsg:
		m.Width, m.Height = msg.Width, msg.Height
		return m, nil
	case DirLoadedMsg:
		t := m.pane(msg.Pane).tabByID(msg.Tab)
		if t == nil {
			return m, nil // tab was closed: stale result (SPEC §3.9 T3)
		}
		t.Cwd = msg.Path
		t.Entries = msg.Entries
		t.Cursor = 0
		t.ScrollOffset = 0
		t.Loading = false
		t.Err = nil
		return m, nil
	case DirLoadErrMsg:
		t := m.pane(msg.Pane).tabByID(msg.Tab)
		if t == nil {
			return m, nil
		}
		t.Loading = false
		t.Err = msg.Err
		t.clampCursor()
		m.StatusText = msg.Err.Error()
		return m, nil
	}
	return m, nil
}

func (m RootModel) handleKey(msg tea.KeyMsg) (RootModel, tea.Cmd) {
	p := m.pane(m.Active)
	t := p.tab()
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "tab":
		if m.Active == LeftPane {
			m.Active = RightPane
		} else {
			m.Active = LeftPane
		}
	case "ctrl+t":
		return m.openTab()
	case "ctrl+w":
		return m.closeTab()
	case "]":
		p.cycleTab(1)
	case "[":
		p.cycleTab(-1)
	case "up", "k":
		t.Cursor--
		t.clampCursor()
	case "down", "j":
		t.Cursor++
		t.clampCursor()
	case " ", "space":
		t.toggleSelected()
		t.Cursor++
		t.clampCursor()
	case "enter", "right", "l":
		if t.Loading || len(t.Entries) == 0 {
			return m, nil
		}
		e := t.Entries[t.Cursor]
		if !e.IsDir {
			return m, nil
		}
		t.Loading = true
		return m, loadDirCmd(m.FS, p.ID, t.ID, e.Path)
	case "backspace", "left", "h":
		if t.Loading {
			return m, nil
		}
		t.Loading = true
		return m, loadDirCmd(m.FS, p.ID, t.ID, filepath.Dir(t.Cwd))
	}
	return m, nil
}

// toggleSelected flips the selection of the entry under the cursor. The map
// is copied first so previous RootModel values are never mutated.
func (t *TabModel) toggleSelected() {
	if len(t.Entries) == 0 {
		return
	}
	sel := make(map[string]struct{}, len(t.Selected)+1)
	for k := range t.Selected {
		sel[k] = struct{}{}
	}
	path := t.Entries[t.Cursor].Path
	if _, ok := sel[path]; ok {
		delete(sel, path)
	} else {
		sel[path] = struct{}{}
	}
	t.Selected = sel
}
