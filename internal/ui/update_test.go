package ui

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/naughtlylab/moon-commander/internal/fs"
)

// SPEC §3.1 — Cursor Bounds.
func TestInvariant1_CursorBounds(t *testing.T) {
	tests := []struct {
		name       string
		entries    []string // left pane contents; nil keeps the fixture
		startCur   int
		keys       []string
		wantCursor int
	}{
		{name: "up at top stays at 0", keys: []string{"up"}, wantCursor: 0},
		{name: "k at top stays at 0", keys: []string{"k", "k"}, wantCursor: 0},
		{name: "down moves 1", keys: []string{"down"}, wantCursor: 1},
		{name: "j moves 1", keys: []string{"j"}, wantCursor: 1},
		{name: "down past end clamps to last", keys: []string{"j", "j", "j", "j", "j", "j"}, wantCursor: 3},
		{name: "down then up", keys: []string{"j", "j", "k"}, wantCursor: 1},
		{name: "space at last entry stays at last", startCur: 3, keys: []string{"space"}, wantCursor: 3},
		{name: "empty pane down stays 0", entries: []string{}, keys: []string{"down", "j"}, wantCursor: 0},
		{name: "empty pane up stays 0", entries: []string{}, keys: []string{"up"}, wantCursor: 0},
		{name: "empty pane space stays 0", entries: []string{}, keys: []string{"space"}, wantCursor: 0},
		{name: "single entry down stays 0", entries: []string{"only.txt"}, keys: []string{"down"}, wantCursor: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, m := fixture()
			if tt.entries != nil {
				m = mustUpdate(m, DirLoadedMsg{Pane: LeftPane, Path: "/home/user", Entries: ents("/home/user", tt.entries...)})
			}
			m.Left.Cursor = tt.startCur
			for _, k := range tt.keys {
				m = mustUpdate(m, key(k))
				assertCursorInvariant(t, m)
			}
			assertEqual(t, "Left.Cursor", m.Left.Cursor, tt.wantCursor)
		})
	}

	t.Run("loading a directory resets/clamps a stale cursor", func(t *testing.T) {
		for _, n := range []int{0, 1, 2} {
			_, m := fixture()
			m.Left.Cursor = 3
			m = mustUpdate(m, DirLoadedMsg{Pane: LeftPane, Path: "/x", Entries: ents("/x", []string{"a", "b"}[:n]...)})
			assertCursorInvariant(t, m)
			assertEqual(t, "Left.Cursor", m.Left.Cursor, 0)
		}
	})

	t.Run("load error keeps cursor valid", func(t *testing.T) {
		_, m := fixture()
		m.Left.Cursor = 2
		m = mustUpdate(m, DirLoadErrMsg{Pane: LeftPane, Path: "/x", Err: errors.New("denied")})
		assertCursorInvariant(t, m)
		assertEqual(t, "Left.Cursor", m.Left.Cursor, 2)
	})
}

// SPEC §3.2 — Directory Sorting (applied by the load command before the
// listing reaches Update).
func TestInvariant2_DirectorySorting(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{name: "dirs first", input: []string{"z.txt", "a/", "m.txt", "b/"}, want: []string{"a", "b", "m.txt", "z.txt"}},
		{name: "case-insensitive", input: []string{"beta", "Alpha", "Zeta/", "alpha2/"}, want: []string{"alpha2", "Zeta", "Alpha", "beta"}},
		{name: "files only", input: []string{"c", "B", "a"}, want: []string{"a", "B", "c"}},
		{name: "dirs only", input: []string{"C/", "b/", "A/"}, want: []string{"A", "b", "C"}},
		{name: "empty", input: []string{}, want: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mfs, m := fixture()
			mfs.AddDir("/home/user/docs", ents("/home/user/docs", tt.input...)...)
			m.Left.Cursor = 0 // docs/

			m, cmd := m.update(key("enter"))
			msg, ok := run(t, cmd).(DirLoadedMsg)
			if !ok {
				t.Fatalf("expected DirLoadedMsg, got %T", run(t, cmd))
			}
			assertEqual(t, "msg entries", entryNames(msg.Entries), tt.want)

			m = mustUpdate(m, msg)
			assertEqual(t, "pane entries", entryNames(m.Left.Entries), tt.want)
		})
	}
}

// SPEC §3.3 — Pane Switching.
func TestInvariant3_PaneSwitching(t *testing.T) {
	tests := []struct {
		name  string
		start ActivePane
		tabs  int
		want  ActivePane
	}{
		{name: "left to right", start: LeftPane, tabs: 1, want: RightPane},
		{name: "right to left", start: RightPane, tabs: 1, want: LeftPane},
		{name: "double tab returns", start: LeftPane, tabs: 2, want: LeftPane},
		{name: "triple tab", start: RightPane, tabs: 3, want: LeftPane},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, m := fixture()
			m.Active = tt.start
			m.Left.Cursor, m.Right.Cursor = 2, 0
			wantLeft, wantRight := m.Left, m.Right
			for i := 0; i < tt.tabs; i++ {
				var cmd tea.Cmd
				m, cmd = m.update(key("tab"))
				if cmd != nil {
					t.Errorf("tab returned non-nil cmd")
				}
			}
			assertEqual(t, "Active", m.Active, tt.want)
			assertEqual(t, "Left", m.Left, wantLeft)
			assertEqual(t, "Right", m.Right, wantRight)
		})
	}
}

// SPEC §3.4 — Directory Traversal.
func TestInvariant4_DirectoryTraversal(t *testing.T) {
	tests := []struct {
		name     string
		active   ActivePane
		cursor   int
		key      string
		wantPath string // "" means no traversal expected
	}{
		{name: "enter on dir", cursor: 0, key: "enter", wantPath: "/home/user/docs"},
		{name: "right on dir", cursor: 1, key: "right", wantPath: "/home/user/src"},
		{name: "l on dir", cursor: 0, key: "l", wantPath: "/home/user/docs"},
		{name: "enter on file is a no-op", cursor: 2, key: "enter"},
		{name: "right on file is a no-op", cursor: 3, key: "right"},
		{name: "enter on file in right pane is a no-op", active: RightPane, key: "enter"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, m := fixture()
			m.Active = tt.active
			m.pane(tt.active).Cursor = tt.cursor
			before := m

			m, cmd := m.update(key(tt.key))

			if tt.wantPath == "" {
				if cmd != nil {
					t.Errorf("expected nil cmd, got one returning %#v", cmd())
				}
				assertEqual(t, "model", m, before)
				return
			}
			p := m.pane(tt.active)
			assertEqual(t, "Loading", p.Loading, true)
			assertEqual(t, "Cwd (unchanged until loaded)", p.Cwd, before.pane(tt.active).Cwd)
			assertEqual(t, "other pane", *m.pane(1 - tt.active), *before.pane(1 - tt.active))

			msg, ok := run(t, cmd).(DirLoadedMsg)
			if !ok {
				t.Fatalf("expected DirLoadedMsg, got %#v", run(t, cmd))
			}
			assertEqual(t, "msg.Pane", msg.Pane, tt.active)
			assertEqual(t, "msg.Path", msg.Path, tt.wantPath)

			m = mustUpdate(m, msg)
			p = m.pane(tt.active)
			assertEqual(t, "Cwd after load", p.Cwd, tt.wantPath)
			assertEqual(t, "Loading after load", p.Loading, false)
			assertEqual(t, "Cursor after load", p.Cursor, 0)
		})
	}

	t.Run("enter on empty pane is a no-op", func(t *testing.T) {
		_, m := fixture()
		m = mustUpdate(m, DirLoadedMsg{Pane: LeftPane, Path: "/home/user/src", Entries: []fs.Entry{}})
		before := m
		m, cmd := m.update(key("enter"))
		if cmd != nil {
			t.Error("expected nil cmd")
		}
		assertEqual(t, "model", m, before)
	})
}

// SPEC §3.5 — Parent Traversal.
func TestInvariant5_ParentTraversal(t *testing.T) {
	tests := []struct {
		name     string
		active   ActivePane
		cwd      string
		key      string
		wantPath string
	}{
		{name: "backspace", cwd: "/home/user", key: "backspace", wantPath: "/home"},
		{name: "left", cwd: "/home/user", key: "left", wantPath: "/home"},
		{name: "h", cwd: "/home/user/docs", key: "h", wantPath: "/home/user"},
		{name: "to root", cwd: "/home", key: "backspace", wantPath: "/"},
		{name: "at root stays root", cwd: "/", key: "backspace", wantPath: "/"},
		{name: "right pane", active: RightPane, cwd: "/tmp", key: "h", wantPath: "/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, m := fixture()
			m.Active = tt.active
			m.pane(tt.active).Cwd = tt.cwd
			other := *m.pane(1 - tt.active)

			m, cmd := m.update(key(tt.key))
			assertEqual(t, "Loading", m.pane(tt.active).Loading, true)
			assertEqual(t, "other pane", *m.pane(1 - tt.active), other)

			msg, ok := run(t, cmd).(DirLoadedMsg)
			if !ok {
				t.Fatalf("expected DirLoadedMsg, got %#v", run(t, cmd))
			}
			assertEqual(t, "msg.Pane", msg.Pane, tt.active)
			assertEqual(t, "msg.Path", msg.Path, tt.wantPath)

			m = mustUpdate(m, msg)
			assertEqual(t, "Cwd after load", m.pane(tt.active).Cwd, tt.wantPath)
			assertEqual(t, "Loading after load", m.pane(tt.active).Loading, false)
		})
	}

	t.Run("parent read failure surfaces DirLoadErrMsg and keeps cwd", func(t *testing.T) {
		mfs, m := fixture()
		boom := errors.New("permission denied")
		mfs.SetError("/home", boom)

		m, cmd := m.update(key("backspace"))
		msg, ok := run(t, cmd).(DirLoadErrMsg)
		if !ok {
			t.Fatalf("expected DirLoadErrMsg, got %#v", run(t, cmd))
		}
		assertEqual(t, "msg", msg, DirLoadErrMsg{Pane: LeftPane, Path: "/home", Err: boom})

		m = mustUpdate(m, msg)
		assertEqual(t, "Cwd", m.Left.Cwd, "/home/user")
		assertEqual(t, "Loading", m.Left.Loading, false)
		if !errors.Is(m.Left.Err, boom) {
			t.Errorf("Err = %v, want %v", m.Left.Err, boom)
		}
		assertEqual(t, "entries kept", len(m.Left.Entries), 4)
	})
}
