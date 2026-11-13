package ui

import "github.com/naughtlylab/moon-commander/internal/fs"

// DirLoadedMsg reports a successful directory listing. Entries are already
// sorted and must not be mutated by the sender after delivery.
type DirLoadedMsg struct {
	Pane    ActivePane
	Tab     TabID
	Path    string
	Entries []fs.Entry
}

// DirLoadErrMsg reports a failed directory listing.
type DirLoadErrMsg struct {
	Pane ActivePane
	Tab  TabID
	Path string
	Err  error
}
