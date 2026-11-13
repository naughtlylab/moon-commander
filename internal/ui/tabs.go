package ui

import (
	"slices"

	tea "github.com/charmbracelet/bubbletea"
)

const lastTabStatus = "Cannot close the last tab"

// openTab inserts a new tab after the active one, at the active tab's
// directory, makes it active and starts loading it (SPEC §3.9 T5).
func (m RootModel) openTab() (RootModel, tea.Cmd) {
	p := m.pane(m.Active)
	nt := newTab(p.NextTabID, p.tab().Cwd)
	p.NextTabID++
	p.Tabs = slices.Insert(p.Tabs, p.ActiveTab+1, nt)
	p.ActiveTab++
	return m, loadDirCmd(m.FS, p.ID, nt.ID, nt.Cwd)
}

// closeTab removes the active tab; the right neighbour (or the new last
// tab) becomes active. The only tab cannot be closed (SPEC §3.9 T6).
func (m RootModel) closeTab() (RootModel, tea.Cmd) {
	p := m.pane(m.Active)
	if len(p.Tabs) == 1 {
		m.StatusText = lastTabStatus
		return m, nil
	}
	p.Tabs = slices.Delete(p.Tabs, p.ActiveTab, p.ActiveTab+1)
	p.ActiveTab = min(p.ActiveTab, len(p.Tabs)-1)
	return m, nil
}

// cycleTab moves the active tab by delta with wrap-around (SPEC §3.9 T7).
func (p *PaneModel) cycleTab(delta int) {
	n := len(p.Tabs)
	p.ActiveTab = ((p.ActiveTab+delta)%n + n) % n
}

// cloneTabs gives the pane its own Tabs backing array so that mutating a
// tab never affects earlier RootModel values that share the slice.
func (p *PaneModel) cloneTabs() {
	p.Tabs = slices.Clone(p.Tabs)
}
