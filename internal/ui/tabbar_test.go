package ui

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

// paneCells returns the left (or right) pane's part of a stripped view line.
func paneCells(m RootModel, line int, pane ActivePane) string {
	r := []rune(viewLines(m)[line])
	if pane == LeftPane {
		return string(r[:m.Width/2])
	}
	return string(r[m.Width/2:])
}

// openTab adds a tab to the active pane and loads path into it.
func openTab(m RootModel, path string) RootModel {
	m = press(m, "ctrl+t")
	return mustUpdate(m, DirLoadedMsg{Pane: m.Active, Tab: m.pane(m.Active).tab().ID, Path: path, Entries: ents(path, "f")})
}

// colorModel returns m with a 256-colour renderer so styles are emitted.
func colorModel(m RootModel) RootModel {
	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(termenv.ANSI256)
	r.SetHasDarkBackground(true)
	m.Styles = NewStyles(r)
	return m
}

// SPEC §6 Tabs — the tab row sits above a full pane border, even with one tab.
func TestTabRowPlacement(t *testing.T) {
	_, m := fixture()
	half := m.Width / 2
	assertEqual(t, "left tab row", paneCells(m, 0, LeftPane), fit(" [user] ", half))
	assertEqual(t, "right tab row", paneCells(m, 0, RightPane), fit(" [tmp] ", m.Width-half))
	assertEqual(t, "left border", paneCells(m, 1, LeftPane), "╭"+strings.Repeat("─", half-2)+"╮")
	if s := viewLines(m)[m.Height-1]; strings.Contains(s, "tab 1/") {
		t.Errorf("single-tab status should have no tab prefix: %q", s)
	}
}

// SPEC §6 Tabs — labels, one-cell gaps, and the no-colour active marker.
func TestTabRowLabels(t *testing.T) {
	tests := []struct {
		name   string
		paths  []string // extra tabs opened after the fixture's /home/user tab
		active int
		want   string
	}{
		{name: "second active", paths: []string{"/home/user/docs"}, active: 1, want: " user   [docs] "},
		{name: "first active", paths: []string{"/home/user/docs"}, active: 0, want: " [user]   docs "},
		{name: "root label", paths: []string{"/"}, active: 1, want: " user   [/] "},
		{name: "long label truncated", paths: []string{"/" + strings.Repeat("x", 40)}, active: 1, want: " user   [" + strings.Repeat("x", 15) + "…] "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, m := fixture()
			for _, p := range tt.paths {
				m = openTab(m, p)
			}
			m.Left.ActiveTab = tt.active
			assertEqual(t, "tab row", paneCells(m, 0, LeftPane), fit(tt.want, m.Width/2))
		})
	}
}

// SPEC §6 Tabs — colours: grey inactive, accent active, muted accent for the
// inactive pane's active tab; no brackets when colour is available.
func TestTabRowColors(t *testing.T) {
	const (
		inactiveBg = "48;5;238"
		activeBg   = "48;5;39"
		mutedBg    = "48;5;24"
		darkFg     = "38;5;16"
	)
	_, m := fixture()
	m = openTab(m, "/home/user/docs") // left: user, docs (active)
	m = colorModel(m)
	row := strings.Split(m.View(), "\n")[0]

	for _, want := range []string{inactiveBg, activeBg, darkFg, mutedBg} {
		if !strings.Contains(row, want) {
			t.Errorf("tab row missing %q: %q", want, row)
		}
	}
	if strings.Contains(ansi.Strip(row), "[") {
		t.Errorf("brackets should only appear without colour: %q", ansi.Strip(row))
	}
	for _, c := range []struct{ bg, label string }{{activeBg, " docs "}, {mutedBg, " tmp "}, {inactiveBg, " user "}} {
		if !styledAs(row, c.bg, c.label) {
			t.Errorf("%q should be styled with %s: %q", c.label, c.bg, row)
		}
	}
	if !styledAs(row, darkFg, " docs ") {
		t.Errorf("active tab should use dark contrast text: %q", row)
	}

	m = press(m, "tab") // focus moves right: left's active tab becomes muted
	row = strings.Split(m.View(), "\n")[0]
	if !styledAs(row, mutedBg, " docs ") || !styledAs(row, activeBg, " tmp ") {
		t.Errorf("after tab, left active tab should be muted and right accented: %q", row)
	}
}

// styledAs reports whether text appears immediately after an SGR sequence
// containing the given colour parameter (e.g. "48;5;39").
func styledAs(row, param, text string) bool {
	re := regexp.MustCompile(`\x1b\[([0-9;]*;)?` + regexp.QuoteMeta(param) + `(;[0-9;]*)?m` + regexp.QuoteMeta(text))
	return re.MatchString(row)
}

func TestTabRowOverflow(t *testing.T) {
	_, m := fixture()
	m = mustUpdate(m, tea.WindowSizeMsg{Width: 60, Height: 10})
	for i := 0; i < 8; i++ {
		m = openTab(m, fmt.Sprintf("/dir/a-rather-long-directory-name-%d", i))
	}
	for _, active := range []int{0, 4, 8} {
		m.Left.ActiveTab = active
		row := paneCells(m, 0, LeftPane)
		if w := ansi.StringWidth(row); w != 30 {
			t.Errorf("active %d: row width %d: %q", active, w, row)
		}
		if !strings.Contains(row, "[") || !strings.Contains(row, "…") {
			t.Errorf("active %d: expected active tab and overflow marker: %q", active, row)
		}
	}
	m.Left.ActiveTab = 0
	if row := paneCells(m, 0, LeftPane); !strings.HasPrefix(row, " [user] ") || !strings.HasSuffix(strings.TrimRight(row, " "), "…") {
		t.Errorf("first tab active: want no left marker, right marker: %q", row)
	}
	m.Left.ActiveTab = 8
	if row := paneCells(m, 0, LeftPane); !strings.HasPrefix(row, "… ") || !strings.Contains(row, "[a-rather-long-") {
		t.Errorf("last tab active: want left marker and visible active tab: %q", row)
	}
}

// SPEC §6 — exact dimensions hold with tab rows at several sizes.
func TestTabRowDimensions(t *testing.T) {
	for _, sz := range []struct{ w, h int }{{80, 24}, {41, 7}, {20, 6}, {120, 30}, {9, 6}} {
		_, m := fixture()
		for i := 0; i < 5; i++ {
			m = openTab(m, fmt.Sprintf("/x/日本語-%d", i))
		}
		m.Active = RightPane
		m = openTab(m, "/var")
		m = colorModel(mustUpdate(m, tea.WindowSizeMsg{Width: sz.w, Height: sz.h}))
		lines := strings.Split(m.View(), "\n")
		if len(lines) != sz.h {
			t.Errorf("%dx%d: %d lines", sz.w, sz.h, len(lines))
		}
		for i, l := range lines {
			if w := lipgloss.Width(l); w != sz.w {
				t.Errorf("%dx%d: line %d width %d: %q", sz.w, sz.h, i, w, ansi.Strip(l))
			}
		}
	}
}

func TestTabStatusPrefix(t *testing.T) {
	_, m := fixture()
	m = openTab(m, "/home/user/docs")
	m = openTab(m, "/tmp")
	m.Left.ActiveTab = 1
	if s := viewLines(m)[m.Height-1]; !strings.Contains(s, "tab 2/3 · 1 items") {
		t.Errorf("status = %q", s)
	}
}
