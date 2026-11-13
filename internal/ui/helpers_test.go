package ui

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/naughtlylab/moon-commander/internal/fs"
)

// key builds a tea.KeyMsg whose String() equals s.
func key(s string) tea.KeyMsg {
	special := map[string]tea.KeyType{
		"enter": tea.KeyEnter, "backspace": tea.KeyBackspace, "tab": tea.KeyTab,
		"up": tea.KeyUp, "down": tea.KeyDown, "left": tea.KeyLeft, "right": tea.KeyRight,
		"space": tea.KeySpace, "ctrl+c": tea.KeyCtrlC, "ctrl+t": tea.KeyCtrlT, "ctrl+w": tea.KeyCtrlW,
	}
	if t, ok := special[s]; ok {
		return tea.KeyMsg{Type: t}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

// ents builds entries under dir; names ending in "/" are directories.
func ents(dir string, specs ...string) []fs.Entry {
	out := make([]fs.Entry, 0, len(specs))
	for _, s := range specs {
		isDir := strings.HasSuffix(s, "/")
		name := strings.TrimSuffix(s, "/")
		out = append(out, fs.Entry{Name: name, Path: filepath.Join(dir, name), IsDir: isDir})
	}
	return out
}

func entryNames(es []fs.Entry) []string {
	out := make([]string, len(es))
	for i, e := range es {
		out[i] = e.Name
	}
	return out
}

// fixture returns a mock FS and a model whose panes are already loaded:
// Left at /home/user (docs/, src/, a.txt, B.txt), Right at /tmp (x.log).
func fixture() (*fs.Mock, RootModel) {
	mfs := fs.NewMock("/home/user")
	mfs.AddDir("/", ents("/", "home/", "tmp/")...)
	mfs.AddDir("/home", ents("/home", "user/")...)
	mfs.AddDir("/home/user", ents("/home/user", "B.txt", "src/", "a.txt", "docs/")...)
	mfs.AddDir("/home/user/docs", ents("/home/user/docs", "notes.md")...)
	mfs.AddDir("/home/user/src")
	mfs.AddDir("/tmp", ents("/tmp", "x.log")...)

	m := NewRootModel(mfs, "/home/user", "/tmp")
	m = mustUpdate(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = mustUpdate(m, DirLoadedMsg{Pane: LeftPane, Path: "/home/user",
		Entries: fs.SortEntries(ents("/home/user", "B.txt", "src/", "a.txt", "docs/"))})
	m = mustUpdate(m, DirLoadedMsg{Pane: RightPane, Path: "/tmp", Entries: ents("/tmp", "x.log")})
	return mfs, m
}

// mustUpdate applies msg and discards the returned command.
func mustUpdate(m RootModel, msg tea.Msg) RootModel {
	m, _ = m.update(msg)
	return m
}

// run executes cmd synchronously and returns its message (nil if cmd is nil).
func run(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		return nil
	}
	return cmd()
}

// assertCursorInvariant checks SPEC §3.1, §3.6 and tab invariants T1/T2 on
// every tab of both panes.
func assertCursorInvariant(t *testing.T, m RootModel) {
	t.Helper()
	rows := m.Height - 5
	if rows < 1 {
		rows = 1
	}
	for _, p := range []PaneModel{m.Left, m.Right} {
		if len(p.Tabs) < 1 || p.ActiveTab < 0 || p.ActiveTab >= len(p.Tabs) {
			t.Errorf("pane %d: T1 violated: %d tabs, ActiveTab %d", p.ID, len(p.Tabs), p.ActiveTab)
			continue
		}
		seen := map[TabID]bool{}
		for _, tb := range p.Tabs {
			if seen[tb.ID] || tb.ID >= p.NextTabID {
				t.Errorf("pane %d: T2 violated: tab ID %d (NextTabID %d, dup %v)", p.ID, tb.ID, p.NextTabID, seen[tb.ID])
			}
			seen[tb.ID] = true
			n := len(tb.Entries)
			maxOff := max(0, n-rows)
			if tb.ScrollOffset < 0 || tb.ScrollOffset > maxOff {
				t.Errorf("pane %d tab %d: ScrollOffset %d out of [0,%d]", p.ID, tb.ID, tb.ScrollOffset, maxOff)
			}
			if n > 0 && (tb.Cursor < tb.ScrollOffset || tb.Cursor >= tb.ScrollOffset+rows) {
				t.Errorf("pane %d tab %d: Cursor %d not visible in [%d,%d)", p.ID, tb.ID, tb.Cursor, tb.ScrollOffset, tb.ScrollOffset+rows)
			}
			if n == 0 && tb.Cursor != 0 {
				t.Errorf("pane %d tab %d: empty but Cursor = %d", p.ID, tb.ID, tb.Cursor)
			}
			if n > 0 && (tb.Cursor < 0 || tb.Cursor >= n) {
				t.Errorf("pane %d tab %d: Cursor %d out of [0,%d)", p.ID, tb.ID, tb.Cursor, n)
			}
		}
	}
}

// tabIDs lists the IDs of a pane's tabs in order.
func tabIDs(p PaneModel) []TabID {
	out := make([]TabID, len(p.Tabs))
	for i, tb := range p.Tabs {
		out[i] = tb.ID
	}
	return out
}

// press applies a sequence of keys, discarding commands.
func press(m RootModel, keys ...string) RootModel {
	for _, k := range keys {
		m = mustUpdate(m, key(k))
	}
	return m
}

func assertEqual(t *testing.T, what string, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s = %#v, want %#v", what, got, want)
	}
}
