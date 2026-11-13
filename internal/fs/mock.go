package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Mock is an in-memory FileSystem for tests. It is safe for concurrent use,
// since tea.Cmd functions may run on background goroutines.
type Mock struct {
	mu   sync.RWMutex
	home string
	dirs map[string][]Entry
	errs map[string]error
}

var _ FileSystem = (*Mock)(nil)

// NewMock creates an empty mock filesystem with the given home directory.
func NewMock(home string) *Mock {
	return &Mock{
		home: home,
		dirs: map[string][]Entry{},
		errs: map[string]error{},
	}
}

// AddDir registers dir with the given children. Each child's Path is set to
// filepath.Join(dir, child.Name). Child directories are not created
// automatically; register them with their own AddDir call if they will be
// listed. Children are returned by ReadDir in insertion order (unsorted).
func (m *Mock) AddDir(dir string, children ...Entry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	dir = filepath.Clean(dir)
	cs := make([]Entry, len(children))
	for i, c := range children {
		c.Path = filepath.Join(dir, c.Name)
		cs[i] = c
	}
	m.dirs[dir] = cs
}

// SetError makes ReadDir and Stat on path return err.
func (m *Mock) SetError(path string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errs[filepath.Clean(path)] = err
}

// ReadDir returns a copy of the registered children of path.
func (m *Mock) ReadDir(path string) ([]Entry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	path = filepath.Clean(path)
	if err, ok := m.errs[path]; ok {
		return nil, err
	}
	cs, ok := m.dirs[path]
	if !ok {
		return nil, notExist("readdir", path)
	}
	out := make([]Entry, len(cs))
	copy(out, cs)
	return out, nil
}

// Stat returns the entry for path, looking first at registered directories
// and then at children of the parent directory.
func (m *Mock) Stat(path string) (Entry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	path = filepath.Clean(path)
	if err, ok := m.errs[path]; ok {
		return Entry{}, err
	}
	for _, c := range m.dirs[filepath.Dir(path)] {
		if c.Path == path {
			return c, nil
		}
	}
	if _, ok := m.dirs[path]; ok {
		return Entry{Name: filepath.Base(path), Path: path, Mode: os.ModeDir | 0o755, IsDir: true}, nil
	}
	return Entry{}, notExist("stat", path)
}

// HomeDir returns the configured home directory.
func (m *Mock) HomeDir() (string, error) {
	return m.home, nil
}

func notExist(op, path string) error {
	return &os.PathError{Op: op, Path: path, Err: fmt.Errorf("mock: %w", os.ErrNotExist)}
}
