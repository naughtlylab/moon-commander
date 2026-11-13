package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/naughtlylab/moon-commander/internal/fs"
)

// loadDirCmd lists path for the given pane tab via fsys off the event loop and reports the result
// as a DirLoadedMsg (entries sorted per SPEC §3.2) or a DirLoadErrMsg.
func loadDirCmd(fsys fs.FileSystem, pane ActivePane, tab TabID, path string) tea.Cmd {
	return func() tea.Msg {
		entries, err := fsys.ReadDir(path)
		if err != nil {
			return DirLoadErrMsg{Pane: pane, Tab: tab, Path: path, Err: err}
		}
		return DirLoadedMsg{Pane: pane, Tab: tab, Path: path, Entries: fs.SortEntries(entries)}
	}
}
