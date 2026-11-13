package fs

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func names(es []Entry) []string {
	out := make([]string, len(es))
	for i, e := range es {
		out[i] = e.Name
	}
	return out
}

func TestSortEntries(t *testing.T) {
	tests := []struct {
		name string
		in   []Entry
		want []string
	}{
		{name: "empty", in: nil, want: []string{}},
		{
			name: "dirs before files",
			in: []Entry{
				{Name: "a.txt"}, {Name: "zdir", IsDir: true}, {Name: "b.txt"}, {Name: "adir", IsDir: true},
			},
			want: []string{"adir", "zdir", "a.txt", "b.txt"},
		},
		{
			name: "case-insensitive within groups",
			in: []Entry{
				{Name: "beta"}, {Name: "Alpha"}, {Name: "Docs", IsDir: true}, {Name: "bin", IsDir: true}, {Name: "charlie"},
			},
			want: []string{"bin", "Docs", "Alpha", "beta", "charlie"},
		},
		{
			name: "case-only differences are ordered deterministically",
			in:   []Entry{{Name: "readme"}, {Name: "README"}, {Name: "Readme"}},
			want: []string{"README", "Readme", "readme"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orig := append([]Entry(nil), tt.in...)
			got := SortEntries(tt.in)
			if !reflect.DeepEqual(names(got), tt.want) {
				t.Errorf("SortEntries() = %v, want %v", names(got), tt.want)
			}
			if !reflect.DeepEqual(tt.in, orig) && len(orig) > 0 {
				t.Errorf("SortEntries mutated its input: %v", names(tt.in))
			}
		})
	}
}

func TestMockFS(t *testing.T) {
	errBoom := errors.New("boom")
	m := NewMock("/home/user")
	m.AddDir("/home/user", Entry{Name: "b.txt"}, Entry{Name: "docs", IsDir: true})
	m.AddDir("/home/user/docs")
	m.SetError("/locked", errBoom)

	tests := []struct {
		name      string
		path      string
		wantNames []string
		wantErr   error
	}{
		{name: "lists entries in insertion order with paths filled", path: "/home/user", wantNames: []string{"b.txt", "docs"}},
		{name: "empty dir", path: "/home/user/docs", wantNames: []string{}},
		{name: "missing dir", path: "/nope", wantErr: os.ErrNotExist},
		{name: "injected error", path: "/locked", wantErr: errBoom},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := m.ReadDir(tt.path)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if !reflect.DeepEqual(names(got), tt.wantNames) {
				t.Errorf("names = %v, want %v", names(got), tt.wantNames)
			}
			for _, e := range got {
				if e.Path != filepath.Join(tt.path, e.Name) {
					t.Errorf("entry %q path = %q", e.Name, e.Path)
				}
			}
		})
	}

	t.Run("returned slice is a copy", func(t *testing.T) {
		a, _ := m.ReadDir("/home/user")
		a[0].Name = "mutated"
		b, _ := m.ReadDir("/home/user")
		if b[0].Name != "b.txt" {
			t.Errorf("mock internal state was mutated via returned slice")
		}
	})

	t.Run("stat", func(t *testing.T) {
		e, err := m.Stat("/home/user/docs")
		if err != nil || !e.IsDir || e.Name != "docs" {
			t.Errorf("Stat dir = %+v, %v", e, err)
		}
		e, err = m.Stat("/home/user/b.txt")
		if err != nil || e.IsDir || e.Name != "b.txt" {
			t.Errorf("Stat file = %+v, %v", e, err)
		}
		if _, err := m.Stat("/home/user/missing"); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("Stat missing err = %v", err)
		}
	})

	t.Run("home", func(t *testing.T) {
		if h, err := m.HomeDir(); err != nil || h != "/home/user" {
			t.Errorf("HomeDir = %q, %v", h, err)
		}
	})
}

func TestOSFS(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	var osfs FileSystem = OS{}

	got, err := osfs.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]Entry{}
	for _, e := range got {
		byName[e.Name] = e
	}
	if e := byName["sub"]; !e.IsDir || e.Path != filepath.Join(dir, "sub") {
		t.Errorf("sub = %+v", e)
	}
	if e := byName["f.txt"]; e.IsDir || e.Size != 5 {
		t.Errorf("f.txt = %+v", e)
	}
	if e, err := osfs.Stat(filepath.Join(dir, "f.txt")); err != nil || e.Size != 5 || e.Name != "f.txt" {
		t.Errorf("Stat = %+v, %v", e, err)
	}
	if _, err := osfs.ReadDir(filepath.Join(dir, "missing")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("ReadDir missing err = %v", err)
	}
}
