package ui

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func tenFiles() []string {
	out := make([]string, 10)
	for i := range out {
		out[i] = fmt.Sprintf("f%02d", i)
	}
	return out
}

// SPEC §3.6 — Scroll Window. Height 8 => visibleRows 4.
func TestInvariant6_ScrollWindow(t *testing.T) {
	type step struct {
		key    string
		height int // if > 0, send a WindowSizeMsg instead of a key
	}
	keys := func(k string, n int) []step {
		out := make([]step, n)
		for i := range out {
			out[i] = step{key: k}
		}
		return out
	}
	cat := func(parts ...[]step) []step {
		var out []step
		for _, p := range parts {
			out = append(out, p...)
		}
		return out
	}
	tests := []struct {
		name       string
		steps      []step
		wantCursor int
		wantOffset int
	}{
		{name: "within window no scroll", steps: keys("j", 3), wantCursor: 3, wantOffset: 0},
		{name: "past window scrolls by one", steps: keys("j", 4), wantCursor: 4, wantOffset: 1},
		{name: "to end", steps: keys("j", 9), wantCursor: 9, wantOffset: 6},
		{name: "past end clamps", steps: keys("j", 15), wantCursor: 9, wantOffset: 6},
		{name: "back up inside window keeps offset", steps: cat(keys("j", 9), keys("k", 3)), wantCursor: 6, wantOffset: 6},
		{name: "back up above window scrolls", steps: cat(keys("j", 9), keys("k", 5)), wantCursor: 4, wantOffset: 4},
		{name: "back to top", steps: cat(keys("j", 9), keys("k", 9)), wantCursor: 0, wantOffset: 0},
		{name: "space advances and scrolls", steps: keys("space", 5), wantCursor: 5, wantOffset: 2},
		{name: "grow window resets offset when all fit", steps: cat(keys("j", 9), []step{{height: 20}}), wantCursor: 9, wantOffset: 0},
		{name: "grow window partially", steps: cat(keys("j", 9), []step{{height: 10}}), wantCursor: 9, wantOffset: 4},
		{name: "shrink window keeps cursor visible", steps: cat(keys("j", 3), []step{{height: 5}}), wantCursor: 3, wantOffset: 3},
		{name: "tiny height still shows one row", steps: cat([]step{{height: 2}}, keys("j", 2)), wantCursor: 2, wantOffset: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, m := fixture()
			m = mustUpdate(m, tea.WindowSizeMsg{Width: 80, Height: 8})
			m = mustUpdate(m, DirLoadedMsg{Pane: LeftPane, Path: "/d", Entries: ents("/d", tenFiles()...)})
			for _, s := range tt.steps {
				if s.height > 0 {
					m = mustUpdate(m, tea.WindowSizeMsg{Width: 80, Height: s.height})
				} else {
					m = mustUpdate(m, key(s.key))
				}
				assertCursorInvariant(t, m)
			}
			assertEqual(t, "Cursor", m.Left.Cursor, tt.wantCursor)
			assertEqual(t, "ScrollOffset", m.Left.ScrollOffset, tt.wantOffset)
		})
	}

	t.Run("directory load resets offset", func(t *testing.T) {
		_, m := fixture()
		m = mustUpdate(m, tea.WindowSizeMsg{Width: 80, Height: 8})
		m = mustUpdate(m, DirLoadedMsg{Pane: LeftPane, Path: "/d", Entries: ents("/d", tenFiles()...)})
		for i := 0; i < 9; i++ {
			m = mustUpdate(m, key("j"))
		}
		m = mustUpdate(m, DirLoadedMsg{Pane: LeftPane, Path: "/e", Entries: ents("/e", tenFiles()...)})
		assertEqual(t, "ScrollOffset", m.Left.ScrollOffset, 0)
		assertCursorInvariant(t, m)
	})

	t.Run("resize applies to inactive pane too", func(t *testing.T) {
		_, m := fixture()
		m = mustUpdate(m, tea.WindowSizeMsg{Width: 80, Height: 8})
		m = mustUpdate(m, DirLoadedMsg{Pane: RightPane, Path: "/d", Entries: ents("/d", tenFiles()...)})
		m.Active = RightPane
		for i := 0; i < 9; i++ {
			m = mustUpdate(m, key("j"))
		}
		m.Active = LeftPane
		m = mustUpdate(m, tea.WindowSizeMsg{Width: 80, Height: 6})
		assertCursorInvariant(t, m)
		assertEqual(t, "Right.ScrollOffset", m.Right.ScrollOffset, 8)
	})
}

// RootModel must satisfy tea.Model and Update must delegate to update.
func TestUpdateWrapper(t *testing.T) {
	var _ tea.Model = RootModel{}
	_, m := fixture()
	want, _ := m.update(key("j"))
	got, _ := m.Update(key("j"))
	assertEqual(t, "Update result", got, tea.Model(want))
}
