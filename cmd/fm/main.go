// Command fm is a dual-pane terminal file manager.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/naughtlylab/moon-commander/internal/fs"
	"github.com/naughtlylab/moon-commander/internal/ui"
)

const usage = "Usage: fm [LEFT_DIR] [RIGHT_DIR]\n"

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "fm:", err)
		os.Exit(1)
	}
	os.Exit(run(os.Args[1:], cwd, os.Stderr, startProgram))
}

// startProgram runs the TUI against the real filesystem.
func startProgram(left, right string) error {
	m := ui.NewRootModel(fs.OS{}, left, right)
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}

// run parses args and launches start, returning the process exit code
// (SPEC §7): 0 on success or -h, 1 for any error.
func run(args []string, cwd string, stderr io.Writer, start func(left, right string) error) int {
	flags := flag.NewFlagSet("fm", flag.ContinueOnError)
	flags.SetOutput(io.Discard) // errors are reported below with the fm: prefix
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stderr, usage)
			return 0
		}
		fmt.Fprintln(stderr, "fm:", err)
		fmt.Fprint(stderr, usage)
		return 1
	}

	left, right, err := resolveDirs(flags.Args(), cwd)
	if err != nil {
		fmt.Fprintln(stderr, "fm:", err)
		fmt.Fprint(stderr, usage)
		return 1
	}
	if err := start(left, right); err != nil {
		fmt.Fprintln(stderr, "fm:", err)
		return 1
	}
	return 0
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
