package ui

import (
	"errors"
	"io"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"

	"github.com/naughtlylab/moon-commander/internal/fs"
)

func viewLines(m RootModel) []string {
	return strings.Split(ansi.Strip(m.View()), "\n")
}

// lineWith returns the index of the first line containing s, or -1.
func lineWith(lines []string, s string) int {
	for i, l := range lines {
		if strings.Contains(l, s) {
			return i
		}
	}
	return -1
}

func TestViewInitializing(t *testing.T) {
	m := NewRootModel(fs.NewMock("/"), "/", "/")
	assertEqual(t, "View", m.View(), "Initializing…")
}

// SPEC §6 — exact dimensions.
func TestViewDimensions(t *testing.T) {
	sizes := []struct{ w, h int }{{80, 24}, {81, 10}, {40, 7}, {120, 30}, {33, 6}}
	for _, sz := range sizes {
		_, m := fixture()
		long := strings.Repeat("very-long-name-", 10)
		m = mustUpdate(m, DirLoadedMsg{Pane: RightPane, Path: "/" + strings.Repeat("deep/", 30),
			Entries: append(ents("/x", long+"/", long), fs.Entry{Name: "日本語のファイル名がとても長い.txt", Size: 1})})
		m = mustUpdate(m, tea.WindowSizeMsg{Width: sz.w, Height: sz.h})
		lines := strings.Split(m.View(), "\n")
		if len(lines) != sz.h {
			t.Errorf("%dx%d: got %d lines", sz.w, sz.h, len(lines))
		}
		for i, l := range lines {
			if w := lipgloss.Width(l); w != sz.w {
				t.Errorf("%dx%d: line %d width %d: %q", sz.w, sz.h, i, w, ansi.Strip(l))
			}
		}
	}
}

func TestViewPaneContent(t *testing.T) {
	_, m := fixture()
	m = mustUpdate(m, DirLoadedMsg{Pane: RightPane, Path: "/tmp", Entries: []fs.Entry{
		{Name: "small", Path: "/tmp/small", Size: 500},
		{Name: "medium", Path: "/tmp/medium", Size: 1234},
		{Name: "big", Path: "/tmp/big", Size: 5 * 1024 * 1024},
	}})
	m = mustUpdate(m, key("j"))
	m = mustUpdate(m, key("j"))
	m = mustUpdate(m, key("space")) // select a.txt
	lines := viewLines(m)

	for _, want := range []string{"/home/user", "/tmp", "docs/", "src/", "B.txt", "<DIR>", "500B", "1.2K", "5.0M", "* a.txt"} {
		if lineWith(lines, want) < 0 {
			t.Errorf("view missing %q:\n%s", want, strings.Join(lines, "\n"))
		}
	}
	if d, a := lineWith(lines, "docs/"), lineWith(lines, "a.txt"); !(d >= 0 && d < a) {
		t.Errorf("expected docs/ above a.txt (got %d, %d)", d, a)
	}
	if lineWith(lines, "* docs/") >= 0 {
		t.Errorf("docs/ should not be marked selected")
	}
}

func TestViewScrollWindow(t *testing.T) {
	_, m := fixture()
	m = mustUpdate(m, tea.WindowSizeMsg{Width: 80, Height: 9})
	m = mustUpdate(m, DirLoadedMsg{Pane: LeftPane, Path: "/d", Entries: ents("/d", tenFiles()...)})
	for i := 0; i < 9; i++ {
		m = mustUpdate(m, key("j"))
	}
	lines := viewLines(m)
	for _, name := range []string{"f06", "f07", "f08", "f09"} {
		if lineWith(lines, name) < 0 {
			t.Errorf("expected %s visible", name)
		}
	}
	for _, name := range []string{"f00", "f05"} {
		if lineWith(lines, name) >= 0 {
			t.Errorf("expected %s scrolled out", name)
		}
	}
}

func TestViewLoadingAndError(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(t *TabModel)
		want    string
		notWant string
	}{
		{name: "loading", mutate: func(t *TabModel) { t.Loading = true }, want: "Loading…", notWant: "docs/"},
		{name: "error", mutate: func(t *TabModel) { t.Err = errors.New("permission denied") }, want: "permission denied", notWant: "docs/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, m := fixture()
			tt.mutate(m.Left.tab())
			lines := viewLines(m)
			if lineWith(lines, tt.want) < 0 {
				t.Errorf("missing %q", tt.want)
			}
			if lineWith(lines, tt.notWant) >= 0 {
				t.Errorf("unexpected %q", tt.notWant)
			}
			if lineWith(lines, "/home/user") < 0 {
				t.Errorf("header should still show cwd")
			}
		})
	}
}

func TestViewTruncation(t *testing.T) {
	_, m := fixture()
	m = mustUpdate(m, tea.WindowSizeMsg{Width: 40, Height: 10})
	m = mustUpdate(m, DirLoadedMsg{Pane: LeftPane, Path: "/a/b/c/d/e/f/g/h/i/j/k/deep/dir",
		Entries: ents("/x", strings.Repeat("n", 100))})
	lines := viewLines(m)
	if h := lines[2]; !strings.HasPrefix(h, "│…") || !strings.Contains(h, "/deep/dir│") {
		t.Errorf("expected left-truncated cwd:\n%s", strings.Join(lines, "\n"))
	}
	if lineWith(lines, "nnn…") < 0 {
		t.Errorf("expected truncated name:\n%s", strings.Join(lines, "\n"))
	}
}

func TestStatusBar(t *testing.T) {
	_, m := fixture()
	last := func(m RootModel) string { l := viewLines(m); return l[len(l)-1] }

	if s := last(m); !strings.Contains(s, "4 items · 0 selected") {
		t.Errorf("status = %q", s)
	}
	m = mustUpdate(m, key("space"))
	if s := last(m); !strings.Contains(s, "4 items · 1 selected") {
		t.Errorf("status after select = %q", s)
	}
	m = mustUpdate(m, key("tab"))
	if s := last(m); !strings.Contains(s, "1 items · 0 selected") {
		t.Errorf("status for right pane = %q", s)
	}
	m.StatusText = "something happened"
	if s := last(m); !strings.Contains(s, "something happened") || strings.Contains(s, "items") {
		t.Errorf("StatusText should override: %q", s)
	}
}

// Cursor highlighting depends on the active pane; checked with a color renderer.
func TestViewCursorStyle(t *testing.T) {
	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(termenv.ANSI256)
	r.SetHasDarkBackground(true)
	const reverse, underline = "\x1b[7m", "\x1b[4m"

	_, m := fixture()
	m.Styles = NewStyles(r)
	m = mustUpdate(m, key("j")) // left cursor on src/
	raw := strings.Split(m.View(), "\n")
	plain := viewLines(m)

	src, docs, xlog := lineWith(plain, "src/"), lineWith(plain, "docs/"), lineWith(plain, "x.log")
	if !strings.Contains(raw[src], reverse) {
		t.Errorf("active cursor row should be reversed: %q", raw[src])
	}
	if strings.Contains(raw[docs], reverse) {
		t.Errorf("non-cursor row should not be reversed: %q", raw[docs])
	}
	if !strings.Contains(raw[xlog], underline) {
		t.Errorf("inactive cursor row should be underlined: %q", raw[xlog])
	}
}

func TestHumanSize(t *testing.T) {
	tests := []struct {
		in   int64
		want string
	}{
		{0, "0B"}, {1023, "1023B"}, {1024, "1.0K"}, {1536, "1.5K"},
		{10 * 1024, "10K"}, {1023 * 1024, "1023K"}, {3 << 30, "3.0G"}, {2 << 40, "2.0T"},
	}
	for _, tt := range tests {
		assertEqual(t, "humanSize", humanSize(tt.in), tt.want)
	}
}
