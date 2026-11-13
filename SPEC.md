# Technical Specification: Modern Dual-Pane TUI File Manager

## 1. System Architecture
The application follows The Elm Architecture (Model-Update-View) via `bubbletea`.

### 1.1 Package Structure
* `cmd/fm/main.go`: Entrypoint, CLI flags, dependency injection, `tea.NewProgram` initialization.
* `internal/fs/`: Filesystem abstraction (`FileSystem` interface, OS implementation, and in-memory mock for testing).
* `internal/ui/`: Bubble Tea `Model`, `Update`, `View`, and sub-components (`Pane`, `StatusBar`, `CommandBar`).
* `internal/keymap/`: Keybinding definitions using `charmbracelet/bubbles/key`.

## 2. Core Data Contracts

### 2.1 Filesystem Abstraction (`internal/fs/fs.go`)
All OS calls must go through this interface to enable deterministic unit testing:

```go
package fs

import (
	"io/fs"
	"time"
)

type Entry struct {
	Name    string
	Path    string
	Size    int64
	Mode    fs.FileMode
	ModTime time.Time
	IsDir   bool
}

type FileSystem interface {
	ReadDir(path string) ([]Entry, error)
	Stat(path string) (Entry, error)
	HomeDir() (string, error)
}
```

### 2.2 Application State (`internal/ui/model.go`)

```go
package ui

import "github.com/naughtlylab/moon-commander/internal/fs"

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
	Styles     Styles // built once by DefaultStyles(); no package-level style vars
}

func (m RootModel) Init() tea.Cmd                          // loads both panes
func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) // thin wrapper over the pure update()
func (m RootModel) View() string
```

### 2.3 Message Protocol (`internal/ui/messages.go`)
Every async operation communicates back to `Update()` strictly via these messages:

```go
package ui

import "github.com/naughtlylab/moon-commander/internal/fs"

type DirLoadedMsg struct {
	Pane    ActivePane
	Path    string
	Entries []fs.Entry
}

type DirLoadErrMsg struct {
	Pane ActivePane
	Path string
	Err  error
}
```

### 2.4 Concurrency & Communication Rules
* **Lock-Free UI State:** Never place `sync.Mutex` inside `RootModel` or `PaneModel`. All state mutations happen sequentially inside `Update()`.
* **Immutable Slice Handoff:** Background `tea.Cmd` goroutines must never mutate an `[]fs.Entry` slice after returning it in `DirLoadedMsg`.
* **Batched Streaming:** Long-running operations (Milestone 2 copy/move/search) must batch progress updates (max 1 message per 16ms) over a channel subscription rather than flooding the Bubble Tea event loop.

## 3. State Machine Invariants (Rules to Test)
1. **Cursor Bounds:** `PaneModel.Cursor` must always be `0` when `len(Entries) == 0`, otherwise clamped between `0` and `len(Entries)-1`.
2. **Directory Sorting:** Directories are always sorted first (alphabetically case-insensitive), followed by files (alphabetically case-insensitive).
3. **Pane Switching (`Tab`):** Toggles `RootModel.Active` between `LeftPane` and `RightPane` without mutating either pane's cursor or directory.
4. **Directory Traversal (`enter` / `right` / `l`):**
   * If the highlighted `Entry.IsDir == true`, set `PaneModel.Loading = true` and return a `tea.Cmd` that calls `FS.ReadDir(entry.Path)`.
   * If `Entry.IsDir == false`, do nothing in Milestone 1.
5. **Parent Traversal (`backspace` / `left` / `h`):**
   * Compute `filepath.Dir(pane.Cwd)`. Set `Loading = true` and dispatch `ReadDir` command for the parent path.
6. **Scroll Window:** `ScrollOffset` is maintained by `Update` (never `View`). With `visibleRows(Height) = max(1, Height-4)` (2 border lines + 1 header + 1 status line), after every message:
   * `ScrollOffset <= Cursor < ScrollOffset + visibleRows`
   * `0 <= ScrollOffset <= max(0, len(Entries) - visibleRows)`
   * Recomputed on cursor moves, directory loads, and `tea.WindowSizeMsg`.
7. **No Navigation While Loading:** Traversal keys (§3.4, §3.5) are no-ops while the active pane has `Loading == true`.

## 4. Keybindings (Milestone 1)
* `up` / `k`: Move cursor up 1 row in active pane.
* `down` / `j`: Move cursor down 1 row in active pane.
* `enter` / `right` / `l`: Enter selected directory.
* `backspace` / `left` / `h`: Go to parent directory.
* `tab`: Switch active pane (`LeftPane` <-> `RightPane`).
* `space`: Toggle selection of current entry in `PaneModel.Selected` and advance cursor down 1.
* `q` / `ctrl+c`: Quit application.

## 5. Implementation Milestones
* [x] **M1.1: Filesystem Interface & In-Memory Mock** (`internal/fs`)
* [x] **M1.2: State Machine & Navigation Tests** (`internal/ui/update_test.go` + `update.go`)
* [x] **M1.3: Dual-Pane Lip Gloss View & Main Entrypoint** (`internal/ui/view.go` + `cmd/fm/main.go`)
* [ ] **M2.0: File Operations (Copy, Move, Delete) with Modal Confirmation & Progress Channels**

## 6. View Layout (M1.3)
* **Dimensions:** Output is exactly `Height` lines, each exactly `Width` cells wide. Left pane outer width = `Width/2`, right pane = remainder. Before the first `tea.WindowSizeMsg` (`Width == 0`), `View()` returns `"Initializing…"`.
* **Panes:** Rounded border. Active pane border uses an accent color; inactive is dimmed.
* **Header:** First inner line shows `Cwd`, left-truncated with `…` if too wide.
* **Rows:** Only the visible window (`ScrollOffset` .. `ScrollOffset+visibleRows`). Each row: selection marker (`*` or space), name (directories suffixed with `/`), right-aligned size column (human-readable for files, e.g. `1.2K`; `<DIR>` for directories). Names truncated with `…` using display width.
* **Cursor Row:** Reverse video in the active pane, underlined in the inactive pane.
* **Loading / Error:** Pane body shows `Loading…` while `Loading`, otherwise the error text when `Err != nil`.
* **Status Bar:** One line. Shows `StatusText` if non-empty; otherwise `<n> items · <k> selected` for the active pane plus short key hints.
* **Files:** `view.go`, `pane_view.go`, `statusbar.go`, `styles.go` (each < 300 lines).

## 7. Entrypoint (`cmd/fm/main.go`)
* Usage: `fm [LEFT_DIR] [RIGHT_DIR]`. `LEFT_DIR` defaults to the current working directory; `RIGHT_DIR` defaults to `LEFT_DIR`.
* `main` resolves paths to absolute (the only OS access outside `fs` commands), injects `fs.OS{}`, and runs `tea.NewProgram(model, tea.WithAltScreen())`.
* A nonexistent start directory is not fatal (the pane displays the error). Program errors are printed to stderr with exit code 1.
* Deferred: `internal/keymap` (§1.1) migration to `bubbles/key` is not part of M1.3.
