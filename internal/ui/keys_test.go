package ui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/naughtlylab/moon-commander/internal/fs"
)

// SPEC §4 — space toggles selection and advances the cursor.
func TestSpaceTogglesSelection(t *testing.T) {
	_, m := fixture()
	path0 := m.Left.Entries[0].Path

	m1 := mustUpdate(m, key("space"))
	if _, ok := m1.Left.Selected[path0]; !ok {
		t.Errorf("expected %q selected", path0)
	}
	assertEqual(t, "cursor after space", m1.Left.Cursor, 1)
	if len(m.Left.Selected) != 0 {
		t.Errorf("update mutated the previous model's Selected map")
	}

	m1.Left.Cursor = 0
	m2 := mustUpdate(m1, key("space"))
	if _, ok := m2.Left.Selected[path0]; ok {
		t.Errorf("expected %q deselected", path0)
	}
	if _, ok := m1.Left.Selected[path0]; !ok {
		t.Errorf("update mutated the previous model's Selected map")
	}

	t.Run("only the active pane is affected", func(t *testing.T) {
		_, m := fixture()
		m.Active = RightPane
		m = mustUpdate(m, key("space"))
		assertEqual(t, "left selected", len(m.Left.Selected), 0)
		assertEqual(t, "right selected", len(m.Right.Selected), 1)
	})
}

// SPEC §4 — q / ctrl+c quit.
func TestQuit(t *testing.T) {
	for _, k := range []string{"q", "ctrl+c"} {
		t.Run(k, func(t *testing.T) {
			_, m := fixture()
			_, cmd := m.update(key(k))
			if _, ok := run(t, cmd).(tea.QuitMsg); !ok {
				t.Errorf("expected tea.QuitMsg, got %#v", run(t, cmd))
			}
		})
	}
}

// Navigation commands are not re-dispatched while a pane is loading, so that
// out-of-order async results cannot race each other.
func TestNavigationIgnoredWhileLoading(t *testing.T) {
	for _, k := range []string{"enter", "l", "backspace", "h"} {
		t.Run(k, func(t *testing.T) {
			_, m := fixture()
			m.Left.Loading = true
			before := m
			m, cmd := m.update(key(k))
			if cmd != nil {
				t.Errorf("expected nil cmd while loading")
			}
			assertEqual(t, "model", m, before)
		})
	}
}

func TestDirMessagesRouteToPane(t *testing.T) {
	boom := errors.New("boom")
	tests := []struct {
		name      string
		msg       tea.Msg
		wantPane  ActivePane
		wantCwd   string
		wantErr   error
		wantNames []string
	}{
		{
			name:     "loaded right",
			msg:      DirLoadedMsg{Pane: RightPane, Path: "/var", Entries: ents("/var", "log/")},
			wantPane: RightPane, wantCwd: "/var", wantNames: []string{"log"},
		},
		{
			name:     "error left",
			msg:      DirLoadErrMsg{Pane: LeftPane, Path: "/root", Err: boom},
			wantPane: LeftPane, wantCwd: "/home/user", wantErr: boom,
			wantNames: []string{"docs", "src", "a.txt", "B.txt"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, m := fixture()
			other := *m.pane(1 - tt.wantPane)
			m.pane(tt.wantPane).Loading = true
			m = mustUpdate(m, tt.msg)
			p := m.pane(tt.wantPane)
			assertEqual(t, "Cwd", p.Cwd, tt.wantCwd)
			assertEqual(t, "Loading", p.Loading, false)
			assertEqual(t, "names", entryNames(p.Entries), tt.wantNames)
			if !errors.Is(p.Err, tt.wantErr) {
				t.Errorf("Err = %v, want %v", p.Err, tt.wantErr)
			}
			if tt.wantErr != nil && m.StatusText == "" {
				t.Errorf("expected StatusText to describe the error")
			}
			assertEqual(t, "other pane", *m.pane(1 - tt.wantPane), other)
		})
	}

	t.Run("successful load clears a previous error", func(t *testing.T) {
		_, m := fixture()
		m = mustUpdate(m, DirLoadErrMsg{Pane: LeftPane, Path: "/root", Err: boom})
		m = mustUpdate(m, DirLoadedMsg{Pane: LeftPane, Path: "/home", Entries: ents("/home", "user/")})
		if m.Left.Err != nil {
			t.Errorf("Err = %v, want nil", m.Left.Err)
		}
	})
}

func TestWindowSize(t *testing.T) {
	_, m := fixture()
	m, cmd := m.update(tea.WindowSizeMsg{Width: 120, Height: 40})
	assertEqual(t, "size", [2]int{m.Width, m.Height}, [2]int{120, 40})
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

func TestUnknownKeyIsNoOp(t *testing.T) {
	_, m := fixture()
	before := m
	m, cmd := m.update(key("x"))
	if cmd != nil {
		t.Error("expected nil cmd")
	}
	assertEqual(t, "model", m, before)
}

func TestInitLoadsBothPanes(t *testing.T) {
	mfs := fs.NewMock("/h")
	mfs.AddDir("/a", ents("/a", "f", "d/")...)
	mfs.AddDir("/b")
	m := NewRootModel(mfs, "/a", "/b")
	if !m.Left.Loading || !m.Right.Loading {
		t.Fatal("panes should start loading")
	}
	batch, ok := run(t, m.Init()).(tea.BatchMsg)
	if !ok {
		t.Fatalf("Init should return a batch")
	}
	got := map[ActivePane]DirLoadedMsg{}
	for _, c := range batch {
		msg := c().(DirLoadedMsg)
		got[msg.Pane] = msg
	}
	assertEqual(t, "left", entryNames(got[LeftPane].Entries), []string{"d", "f"})
	assertEqual(t, "right path", got[RightPane].Path, "/b")
}

// SPEC §3.8 — selections persist across directory changes and load errors.
func TestSelectionPersistsAcrossDirs(t *testing.T) {
	_, m := fixture()
	m = mustUpdate(m, key("j"))
	m = mustUpdate(m, key("j"))
	m = mustUpdate(m, key("space")) // select /home/user/a.txt
	m.Left.Cursor = 0               // docs/

	m, cmd := m.update(key("enter"))
	m = mustUpdate(m, run(t, cmd))
	assertEqual(t, "Cwd", m.Left.Cwd, "/home/user/docs")
	if _, ok := m.Left.Selected["/home/user/a.txt"]; !ok {
		t.Fatalf("selection lost after directory change: %v", m.Left.Selected)
	}
	m = mustUpdate(m, key("space")) // select /home/user/docs/notes.md
	m = mustUpdate(m, DirLoadErrMsg{Pane: LeftPane, Path: "/root", Err: errors.New("denied")})
	assertEqual(t, "selected count", len(m.Left.Selected), 2)

	m.StatusText = ""
	lines := viewLines(m)
	if s := lines[len(lines)-1]; !strings.Contains(s, "1 items · 2 selected") {
		t.Errorf("status should show pane-wide total: %q", s)
	}
}
