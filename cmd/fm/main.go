// Command fm is a dual-pane terminal file manager.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/naughtlylab/moon-commander/internal/fs"
	"github.com/naughtlylab/moon-commander/internal/ui"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [LEFT_DIR] [RIGHT_DIR]\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	flag.Parse()

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "fm:", err)
		os.Exit(1)
	}
	left, right, err := resolveDirs(flag.Args(), cwd)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fm:", err)
		flag.Usage()
		os.Exit(2)
	}

	m := ui.NewRootModel(fs.OS{}, left, right)
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "fm:", err)
		os.Exit(1)
	}
}

// resolveDirs maps positional args to absolute start directories (SPEC §7).
// It is pure: relative paths are resolved against cwd.
func resolveDirs(args []string, cwd string) (left, right string, err error) {
	if len(args) > 2 {
		return "", "", errors.New("at most two directories may be given")
	}
	abs := func(p string) string {
		if !filepath.IsAbs(p) {
			p = filepath.Join(cwd, p)
		}
		return filepath.Clean(p)
	}
	left = filepath.Clean(cwd)
	if len(args) > 0 {
		left = abs(args[0])
	}
	right = left
	if len(args) > 1 {
		right = abs(args[1])
	}
	return left, right, nil
}
