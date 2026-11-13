package ui

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// withTabs returns the fixture with n tabs in the left pane (IDs 0..n-1),
// each loaded at /home/user, and ActiveTab set to active.
func withTabs(n, active int) RootModel {
	_, m := fixture()
	for i := 1; i < n; i++ {
		var cmd tea.Cmd
		m, cmd = m.update(key("ctrl+t"))
		m = mustUpdate(m, cmd())
	}
	m.Left.ActiveTab = active
	return m
}

// SPEC §3.9 T5 — new tab.
func TestTabNew(t *testing.T) {
	tests := []struct {
		name       string
		tabs       int
		active     int
		wantIDs    []TabID
		wantActive int
	}{
		{name: "from single tab", tabs: 1, active: 0, wantIDs: []TabID{0, 1}, wantActive: 1},
		{name: "inserted after active, not at end", tabs: 3, active: 0, wantIDs: []TabID{0, 3, 1, 2}, wantActive: 1},
		{name: "after last tab", tabs: 3, active: 2, wantIDs: []TabID{0, 1, 2, 3}, wantActive: 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := withTabs(tt.tabs, tt.active)
			m.Left.tab().Cursor = 2
			m = mustUpdate(m, key("space")) // give the current tab a selection
			before := m

			m, cmd := m.update(key("ctrl+t"))
			assertCursorInvariant(t, m)
			assertEqual(t, "IDs", tabIDs(m.Left), tt.wantIDs)
			assertEqual(t, "ActiveTab", m.Left.ActiveTab, tt.wantActive)
			assertEqual(t, "NextTabID", m.Left.NextTabID, TabID(tt.tabs+1))

			nt := m.Left.tab()
			assertEqual(t, "new Cwd", nt.Cwd, before.Left.tab().Cwd)
			assertEqual(t, "new Loading", nt.Loading, true)
			assertEqual(t, "new Selected", len(nt.Selected), 0)
			assertEqual(t, "old tab untouched", m.Left.Tabs[tt.active], before.Left.Tabs[tt.active])
			assertEqual(t, "right pane untouched", m.Right, before.Right)

			msg, ok := run(t, cmd).(DirLoadedMsg)
			if !ok {
				t.Fatalf("expected DirLoadedMsg, got %#v", run(t, cmd))
			}
			assertEqual(t, "msg", [3]any{msg.Pane, msg.Tab, msg.Path}, [3]any{LeftPane, TabID(tt.tabs), "/home/user"})
			m = mustUpdate(m, msg)
			assertEqual(t, "loaded", m.Left.tab().Loading, false)
			assertEqual(t, "entries", entryNames(m.Left.tab().Entries), []string{"docs", "src", "a.txt", "B.txt"})
		})
	}

	t.Run("acts on the active pane only", func(t *testing.T) {
		_, m := fixture()
		m.Active = RightPane
		before := m
		m, cmd := m.update(key("ctrl+t"))
		assertEqual(t, "left untouched", m.Left, before.Left)
		assertEqual(t, "right tabs", len(m.Right.Tabs), 2)
		assertEqual(t, "right cwd", m.Right.tab().Cwd, "/tmp")
		assertEqual(t, "msg pane", run(t, cmd).(DirLoadedMsg).Pane, RightPane)
	})
}

// SPEC §3.9 T7 — switching tabs.
func TestTabSwitch(t *testing.T) {
	tests := []struct {
		name       string
		tabs       int
		active     int
		keys       []string
		wantActive int
	}{
		{name: "next", tabs: 3, active: 0, keys: []string{"]"}, wantActive: 1},
		{name: "next wraps", tabs: 3, active: 2, keys: []string{"]"}, wantActive: 0},
		{name: "prev", tabs: 3, active: 2, keys: []string{"["}, wantActive: 1},
		{name: "prev wraps", tabs: 3, active: 0, keys: []string{"["}, wantActive: 2},
		{name: "full cycle", tabs: 3, active: 1, keys: []string{"]", "]", "]"}, wantActive: 1},
		{name: "single tab next is no-op", tabs: 1, active: 0, keys: []string{"]"}, wantActive: 0},
		{name: "single tab prev is no-op", tabs: 1, active: 0, keys: []string{"["}, wantActive: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := withTabs(tt.tabs, tt.active)
			before := m
			for _, k := range tt.keys {
				var cmd tea.Cmd
				m, cmd = m.update(key(k))
				if cmd != nil {
					t.Errorf("%s returned a command", k)
				}
			}
			assertEqual(t, "ActiveTab", m.Left.ActiveTab, tt.wantActive)
			assertEqual(t, "tabs untouched", m.Left.Tabs, before.Left.Tabs)
			assertEqual(t, "right untouched", m.Right, before.Right)
		})
	}

	t.Run("each tab keeps its own cursor", func(t *testing.T) {
		m := withTabs(2, 0)
		m = press(m, "j", "j")
		m = press(m, "]", "j", "[")
		assertEqual(t, "tab 0 cursor", m.Left.tab().Cursor, 2)
		m = press(m, "]")
		assertEqual(t, "tab 1 cursor", m.Left.tab().Cursor, 1)
	})
}

// SPEC §3.9 T6 — closing tabs.
func TestTabClose(t *testing.T) {
	tests := []struct {
		name       string
		tabs       int
		active     int
		wantIDs    []TabID
		wantActive int
		wantStatus string
	}{
		{name: "middle selects right neighbour", tabs: 3, active: 1, wantIDs: []TabID{0, 2}, wantActive: 1},
		{name: "first", tabs: 3, active: 0, wantIDs: []TabID{1, 2}, wantActive: 0},
		{name: "last selects new last", tabs: 3, active: 2, wantIDs: []TabID{0, 1}, wantActive: 1},
		{name: "only tab is kept", tabs: 1, active: 0, wantIDs: []TabID{0}, wantActive: 0, wantStatus: "Cannot close the last tab"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := withTabs(tt.tabs, tt.active)
			before := m
			m, cmd := m.update(key("ctrl+w"))
			if cmd != nil {
				t.Error("close returned a command")
			}
			assertCursorInvariant(t, m)
			assertEqual(t, "IDs", tabIDs(m.Left), tt.wantIDs)
			assertEqual(t, "ActiveTab", m.Left.ActiveTab, tt.wantActive)
			assertEqual(t, "NextTabID", m.Left.NextTabID, before.Left.NextTabID)
			assertEqual(t, "StatusText", m.StatusText, tt.wantStatus)
			assertEqual(t, "right untouched", m.Right, before.Right)
		})
	}
}

// SPEC §3.9 T2/T3 — IDs are never reused and stale results are dropped.
func TestTabStaleResults(t *testing.T) {
	t.Run("IDs never reused", func(t *testing.T) {
		m := press(withTabs(1, 0), "ctrl+t", "ctrl+w", "ctrl+t")
		assertEqual(t, "IDs", tabIDs(m.Left), []TabID{0, 2})
	})

	msgs := map[string]func(cmd tea.Cmd) tea.Msg{
		"loaded": func(cmd tea.Cmd) tea.Msg { return cmd() },
		"error": func(tea.Cmd) tea.Msg {
			return DirLoadErrMsg{Pane: LeftPane, Tab: 1, Path: "/home/user", Err: errors.New("denied")}
		},
	}
	for name, mk := range msgs {
		t.Run("result for closed tab is ignored: "+name, func(t *testing.T) {
			_, m := fixture()
			m = press(m, "j")                // tab 0: cursor 1, so a misrouted load is visible
			m = withLoadedNewTab(t, m)       // tab 1 at /home/user, cursor on docs/
			m, cmd := m.update(key("l"))     // tab 1 starts loading /home/user/docs...
			m = mustUpdate(m, key("ctrl+w")) // ...and is closed
			before := m
			m = mustUpdate(m, mk(cmd))
			assertEqual(t, "model", m, before)
		})
	}
}

// withLoadedNewTab opens a left-pane tab and completes its initial load.
func withLoadedNewTab(t *testing.T, m RootModel) RootModel {
	t.Helper()
	m, cmd := m.update(key("ctrl+t"))
	return mustUpdate(m, run(t, cmd))
}

// SPEC §3.9 T4 — results for inactive tabs are applied in the background.
func TestTabBackgroundLoad(t *testing.T) {
	_, m := fixture()
	m, cmd := m.update(key("ctrl+t"))
	m = mustUpdate(m, key("[")) // back to tab 0 before the load completes
	tab0 := m.Left.Tabs[0]

	m = mustUpdate(m, cmd())
	assertEqual(t, "ActiveTab", m.Left.ActiveTab, 0)
	assertEqual(t, "tab 0 untouched", m.Left.Tabs[0], tab0)
	bg := m.Left.Tabs[1]
	assertEqual(t, "bg Loading", bg.Loading, false)
	assertEqual(t, "bg entries", len(bg.Entries), 4)
}

// SPEC §3.9 T8 — navigation acts on, and is routed to, the active tab.
func TestTabNavigationRouting(t *testing.T) {
	m := withTabs(2, 1)
	tab0 := m.Left.Tabs[0]
	m.Left.tab().Cursor = 0 // docs/

	m, cmd := m.update(key("enter"))
	msg := run(t, cmd).(DirLoadedMsg)
	assertEqual(t, "msg.Tab", msg.Tab, TabID(1))
	m = mustUpdate(m, msg)
	assertEqual(t, "tab 1 cwd", m.Left.tab().Cwd, "/home/user/docs")
	assertEqual(t, "tab 0 untouched", m.Left.Tabs[0], tab0)

	t.Run("loading guard is per tab", func(t *testing.T) {
		m := withTabs(2, 0)
		m.Left.Tabs[1].Loading = true
		_, cmd := m.update(key("backspace"))
		if cmd == nil {
			t.Error("active tab is not loading, navigation should proceed")
		}
	})
}

// SPEC §3.6/§3.9 — resizing re-syncs scroll for every tab.
func TestTabResizeSyncsAllTabs(t *testing.T) {
	m := withTabs(2, 0)
	m = mustUpdate(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	for i := range m.Left.Tabs {
		m.Left.ActiveTab = i
		m = mustUpdate(m, DirLoadedMsg{Pane: LeftPane, Tab: m.Left.tab().ID, Path: "/d", Entries: ents("/d", tenFiles()...)})
		m = press(m, "j", "j", "j", "j", "j", "j", "j", "j", "j")
	}
	m = mustUpdate(m, tea.WindowSizeMsg{Width: 80, Height: 7})
	assertCursorInvariant(t, m)
	for i, tb := range m.Left.Tabs {
		assertEqual(t, "offset of tab", [2]int{i, tb.ScrollOffset}, [2]int{i, 8})
	}
}

// Update must never mutate the model it was called on (SPEC §3: pure
// transitions). Compare against an independently built identical fixture.
func TestUpdateDoesNotMutateReceiver(t *testing.T) {
	steps := []tea.Msg{
		key("j"), key("space"), key("ctrl+t"), key("]"), key("["), key("ctrl+w"),
		tea.WindowSizeMsg{Width: 50, Height: 6},
		DirLoadedMsg{Pane: LeftPane, Tab: 0, Path: "/d", Entries: ents("/d", tenFiles()...)},
		DirLoadErrMsg{Pane: LeftPane, Tab: 0, Path: "/x", Err: errors.New("x")},
	}
	for _, msg := range steps {
		m, ref := withTabs(3, 1), withTabs(3, 1)
		_, _ = m.update(msg)
		assertEqual(t, "receiver after update", m, ref)
	}
}
