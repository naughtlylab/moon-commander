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

type TabID int

// TabModel is an independent browsing context within a pane.
type TabModel struct {
	ID           TabID
	Cwd          string
	Entries      []fs.Entry
	Cursor       int // Index in Entries (0 <= Cursor < len(Entries))
	ScrollOffset int
	Loading      bool
	Err          error
	Selected     map[string]struct{} // Multi-select paths
}

type PaneModel struct {
	ID        ActivePane
	Tabs      []TabModel // len(Tabs) >= 1, always
	ActiveTab int        // index into Tabs
	NextTabID TabID      // monotonic per pane; IDs are never reused
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
	Tab     TabID
	Path    string
	Entries []fs.Entry
}

type DirLoadErrMsg struct {
	Pane ActivePane
	Tab  TabID
	Path string
	Err  error
}
```

### 2.4 Concurrency & Communication Rules
* **Lock-Free UI State:** Never place `sync.Mutex` inside `RootModel`, `PaneModel` or `TabModel`. All state mutations happen sequentially inside `Update()`.
* **Immutable Slice Handoff:** Background `tea.Cmd` goroutines must never mutate an `[]fs.Entry` slice after returning it in `DirLoadedMsg`.
* **Batched Streaming:** Long-running operations (Milestone 2 copy/move/search) must batch progress updates (max 1 message per 16ms) over a channel subscription rather than flooding the Bubble Tea event loop.

## 3. State Machine Invariants (Rules to Test)
Invariants 1–8 apply to each tab (`TabModel`); key actions act on the active tab of the active pane.

1. **Cursor Bounds:** `TabModel.Cursor` must always be `0` when `len(Entries) == 0`, otherwise clamped between `0` and `len(Entries)-1`.
2. **Directory Sorting:** Directories are always sorted first (alphabetically case-insensitive), followed by files (alphabetically case-insensitive).
3. **Pane Switching (`Tab`):** Toggles `RootModel.Active` between `LeftPane` and `RightPane` without mutating either pane's cursor or directory.
4. **Directory Traversal (`enter` / `right` / `l`):**
   * If the highlighted `Entry.IsDir == true`, set `TabModel.Loading = true` and return a `tea.Cmd` that calls `FS.ReadDir(entry.Path)`.
   * If `Entry.IsDir == false`, do nothing in Milestone 1.
5. **Parent Traversal (`backspace` / `left` / `h`):**
   * Compute `filepath.Dir(tab.Cwd)`. Set `Loading = true` and dispatch `ReadDir` command for the parent path.
6. **Scroll Window:** `ScrollOffset` is maintained by `Update` (never `View`). With `visibleRows(Height) = max(1, Height-5)` (tab row + 2 border lines + 1 header + 1 status line), after every message:
   * `ScrollOffset <= Cursor < ScrollOffset + visibleRows`
   * `0 <= ScrollOffset <= max(0, len(Entries) - visibleRows)`
   * Recomputed on cursor moves, directory loads, and `tea.WindowSizeMsg`, for every tab in both panes.
7. **No Navigation While Loading:** Traversal keys (§3.4, §3.5) are no-ops while the active tab has `Loading == true`.
8. **Selection Persistence:** `TabModel.Selected` is per tab, keyed by absolute path and is NOT cleared by directory changes (`DirLoadedMsg`) or load errors, so selections can be gathered across directories for Milestone 2 operations.
9. **Tabs:** Each pane holds an ordered list of tabs and shows only its active tab. `tab` still switches between the left and right pane.
   * **T1 Tab bounds:** every pane has `len(Tabs) >= 1` and `0 <= ActiveTab < len(Tabs)`.
   * **T2 Stable identity:** tab IDs are unique within a pane and never reused (`NextTabID` only increases). Async results are routed by `(Pane, Tab)`, never by index.
   * **T3 Stale results dropped:** a `DirLoadedMsg`/`DirLoadErrMsg` whose `Tab` no longer exists (the tab was closed) is ignored and leaves the model unchanged.
   * **T4 Background tabs:** a result for an existing but inactive tab is applied to that tab. Loading completes in the background.
   * **T5 New tab:** inserted at `ActiveTab+1` and becomes active. It opens at the current tab's `Cwd` with `Loading = true` and an empty selection, and a `ReadDir` command is dispatched for its ID. Other tabs are not mutated.
   * **T6 Close tab:** removes the active tab. The new active tab is the one to the right, or the new last tab if the closed tab was last. Closing a pane's only tab is a no-op that sets `StatusText = "Cannot close the last tab"`. In-flight loads for the closed tab are dropped (T3).
   * **T7 Switch tab:** next/previous move `ActiveTab` by ±1 with wrap-around and do not mutate any `TabModel`. With one tab it is a no-op.
   * **T8 Scoping:** tab actions affect only the active pane. Invariants §3.1–§3.8 apply per tab.

## 4. Keybindings (Milestone 1)
* `up` / `k`: Move cursor up 1 row in active pane.
* `down` / `j`: Move cursor down 1 row in active pane.
* `enter` / `right` / `l`: Enter selected directory.
* `backspace` / `left` / `h`: Go to parent directory.
* `tab`: Switch active pane (`LeftPane` <-> `RightPane`).
* `]` / `[`: Next / previous tab in the active pane (wraps around).
* `ctrl+t`: New tab in the active pane, opened at the current tab's directory.
* `ctrl+w`: Close the active tab (not the pane's last tab).
* `space`: Toggle selection of current entry in the active tab's `Selected` and advance cursor down 1.
* `q` / `ctrl+c`: Quit application.

**Why tabs don't use `ctrl+tab` / `alt+tab`:** Bubble Tea v1 has no `ctrl+tab` key. Legacy terminal input encodes it the same as `Tab`, and the extended keyboard protocols that can distinguish it (kitty, `modifyOtherKeys`) are not parsed by Bubble Tea v1. Windows Terminal and the VS Code terminal also bind `ctrl+tab` themselves. `alt+tab` is recognised by Bubble Tea (`"alt+tab"`), but the OS window switcher (Windows and most Linux desktops) intercepts it before it reaches the terminal. `]` and `[` are plain keys that every terminal delivers unchanged.

## 5. Implementation Milestones
* [x] **M1.1: Filesystem Interface & In-Memory Mock** (`internal/fs`)
* [x] **M1.2: State Machine & Navigation Tests** (`internal/ui/update_test.go` + `update.go`)
* [x] **M1.3: Dual-Pane Lip Gloss View & Main Entrypoint** (`internal/ui/view.go` + `cmd/fm/main.go`)
* [x] **M1.4: Pane Tabs** (§3.9; `internal/ui/tabs.go` + `tabs_test.go`)
* [ ] **M2.0: File Operations (Copy, Move, Delete) with Modal Confirmation & Progress Channels**

## 6. View Layout (M1.3)
* **Dimensions:** Output is exactly `Height` lines, each exactly `Width` cells wide. Left pane outer width = `Width/2`, right pane = remainder. Before the first `tea.WindowSizeMsg` (`Width == 0`), `View()` returns `"Initializing…"`.
  * **Limitation:** The exact-dimension guarantee holds only for `Height >= 6` (tab row + 2 border lines + header + 1 entry row + status bar). Smaller terminals render more lines than `Height`; no special handling is provided.
* **Panes:** Rounded border. Active pane border uses an accent color; inactive is dimmed.
* **Header:** First inner line shows `Cwd`, left-truncated with `…` if too wide.
* **Rows:** Only the visible window (`ScrollOffset` .. `ScrollOffset+visibleRows`). Each row: selection marker (`*` or space), name (directories suffixed with `/`), right-aligned size column (human-readable for files, e.g. `1.2K`; `<DIR>` for directories). Names truncated with `…` using display width.
* **Cursor Row:** Reverse video in the active pane, underlined in the inactive pane.
* **Loading / Error:** Pane body shows `Loading…` while `Loading`, otherwise the error text when `Err != nil`.
* **Status Bar:** One line. Shows `StatusText` if non-empty; otherwise `<n> items · <k> selected` for the active pane plus short key hints. `<n>` is `len(Entries)` of the current directory; `<k>` is `len(Selected)`, the pane-wide total across all directories (§3.8).
* **Tabs:**
  * **Placement:** each pane has a one-line **tab row directly above its bordered box**, spanning the pane's outer width. The tab row is always shown, including when the pane has a single tab.
  * **Tab boxes:** each tab is a one-line rectangular block: the label padded with one space on each side (` docs `) and filled with a background colour. Boxes are separated by one blank cell. The rest of the row is blank (no background).
  * **Label:** the tab's `filepath.Base(Cwd)` (`/` for the root), truncated with `…` to at most 16 cells.
  * **Colours** (all part of `Styles`):
    | Tab | Background | Text |
    |---|---|---|
    | Inactive tab | disabled grey (`238`) | light contrast text (`252`) |
    | Active tab, active pane | accent (`39`, the active border colour) | dark contrast text (`16`), bold |
    | Active tab, inactive pane | muted accent (`24`) | light contrast text (`252`), bold |
  * **No-colour terminals:** when the renderer's colour profile has no colour (e.g. `NO_COLOR`, dumb terminals, test output), backgrounds are not visible, so the active tab's label is additionally wrapped in brackets (` [docs] `). With colour, no brackets are drawn.
  * **Overflow:** if the boxes don't fit, a window of whole boxes around the active tab is shown, with a plain `…` (no background) at each clipped edge.
  * **Height:** the tab row costs one line in both panes (§3.6). The pane box keeps its full rounded border, including the top.
  * The status bar prefixes `tab <i>/<n> · ` when the active pane has more than one tab (unchanged).
* **Files:** `view.go`, `pane_view.go`, `tabbar.go`, `statusbar.go`, `styles.go` (each < 300 lines).

## 7. Entrypoint (`cmd/fm/main.go`)
* Usage: `fm [LEFT_DIR] [RIGHT_DIR]`. `LEFT_DIR` defaults to the current working directory; `RIGHT_DIR` defaults to `LEFT_DIR`.
* `main` resolves paths to absolute (the only OS access outside `fs` commands), injects `fs.OS{}`, and runs `tea.NewProgram(model, tea.WithAltScreen())`.
* A nonexistent start directory is not fatal (the pane displays the error).
* **Exit codes:** `0` on normal quit and for `-h`/`-help`. `1` for every error, including usage errors (too many arguments, unknown flags) and program errors. Errors are printed to stderr prefixed with `fm:`; usage errors also print the usage text.
* `main()` delegates to a testable `run(args, cwd, stderr, start) int`, where `start` launches the Bubble Tea program and is injected so tests never open a terminal.
* Deferred: `internal/keymap` (§1.1) migration to `bubbles/key` is not part of M1.3.

## 8. Known Issues
* **Startup key drop (deferred):** Keys sent in a rapid burst immediately after launch may be lost (observed under tmux); keys typed after the first frame are handled normally. The suspected cause is terminal capability/background queries during Bubble Tea/termenv startup consuming input. Unconfirmed; revisit after M2.0.
