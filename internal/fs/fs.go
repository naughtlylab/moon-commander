// Package fs abstracts all filesystem access so UI logic can be tested
// deterministically against an in-memory mock.
package fs

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Entry describes a single directory entry.
type Entry struct {
	Name    string
	Path    string
	Size    int64
	Mode    fs.FileMode
	ModTime time.Time
	IsDir   bool
}

// FileSystem is the only gateway to disk I/O used by the application.
type FileSystem interface {
	ReadDir(path string) ([]Entry, error)
	Stat(path string) (Entry, error)
	HomeDir() (string, error)
}

// OS is the real FileSystem backed by the operating system.
type OS struct{}

var _ FileSystem = OS{}

// ReadDir lists path. Entries whose metadata cannot be read (e.g. a file
// removed mid-listing) are still returned with name, path and type only.
func (OS) ReadDir(path string) ([]Entry, error) {
	des, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(des))
	for _, de := range des {
		e := Entry{
			Name:  de.Name(),
			Path:  filepath.Join(path, de.Name()),
			Mode:  de.Type(),
			IsDir: de.IsDir(),
		}
		if info, err := de.Info(); err == nil {
			e.Size = info.Size()
			e.Mode = info.Mode()
			e.ModTime = info.ModTime()
		}
		out = append(out, e)
	}
	return out, nil
}

// Stat returns metadata for path.
func (OS) Stat(path string) (Entry, error) {
	info, err := os.Stat(path)
	if err != nil {
		return Entry{}, err
	}
	return fromFileInfo(path, info), nil
}

// HomeDir returns the current user's home directory.
func (OS) HomeDir() (string, error) {
	return os.UserHomeDir()
}

func fromFileInfo(path string, info fs.FileInfo) Entry {
	return Entry{
		Name:    info.Name(),
		Path:    path,
		Size:    info.Size(),
		Mode:    info.Mode(),
		ModTime: info.ModTime(),
		IsDir:   info.IsDir(),
	}
}

// SortEntries returns a new slice with directories first, then files, each
// group ordered case-insensitively by name. Names equal ignoring case fall
// back to a byte-wise comparison so the ordering is total and deterministic.
// The input slice is not modified.
func SortEntries(in []Entry) []Entry {
	out := make([]Entry, len(in))
	copy(out, in)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.IsDir != b.IsDir {
			return a.IsDir
		}
		la, lb := strings.ToLower(a.Name), strings.ToLower(b.Name)
		if la != lb {
			return la < lb
		}
		return a.Name < b.Name
	})
	return out
}
