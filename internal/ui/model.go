// Package ui contains the Bubble Tea model and its pure state transitions.
package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/naughtlylab/moon-commander/internal/fs"
)

type ActivePane int

const (
	LeftPane ActivePane = iota
	RightPane
)

type AppMode int

const (
	ModeNormal AppMode = iota
	ModeFilter
	ModeConfirmModal
)

type PaneModel struct {
	ID           ActivePane
	Cwd          string
	Entries      []fs.Entry
	Cursor       int // Index in Entries (0 <= Cursor < len(Entries))
	ScrollOffset int
	Loading      bool
	Err          error
	Selected     map[string]struct{} // Multi-select paths
}

type RootModel struct {
	FS         fs.FileSystem
	Left       PaneModel
	Right      PaneModel
	Active     ActivePane
	Mode       AppMode
	Width      int
	Height     int
	StatusText string
	Styles     Styles
}

// NewRootModel builds the initial model. Both panes start in the Loading
// state; Init dispatches the corresponding ReadDir commands.
func NewRootModel(fsys fs.FileSystem, leftDir, rightDir string) RootModel {
	return RootModel{
		FS:     fsys,
		Left:   newPane(LeftPane, leftDir),
		Right:  newPane(RightPane, rightDir),
		Active: LeftPane,
		Mode:   ModeNormal,
		Styles: DefaultStyles(),
	}
}

func newPane(id ActivePane, dir string) PaneModel {
	return PaneModel{ID: id, Cwd: dir, Loading: true, Selected: map[string]struct{}{}}
}

// Init loads both panes' starting directories.
func (m RootModel) Init() tea.Cmd {
	return tea.Batch(
		loadDirCmd(m.FS, LeftPane, m.Left.Cwd),
		loadDirCmd(m.FS, RightPane, m.Right.Cwd),
	)
}

// Update implements tea.Model by delegating to the pure update function.
func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m.update(msg)
}

// pane returns a pointer to the pane with the given ID within m.
func (m *RootModel) pane(id ActivePane) *PaneModel {
	if id == RightPane {
		return &m.Right
	}
	return &m.Left
}

// clampCursor enforces State Machine Invariant 1.
func (p *PaneModel) clampCursor() {
	switch {
	case len(p.Entries) == 0, p.Cursor < 0:
		p.Cursor = 0
	case p.Cursor >= len(p.Entries):
		p.Cursor = len(p.Entries) - 1
	}
}
