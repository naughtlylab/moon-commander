package main

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestResolveDirs(t *testing.T) {
	const cwd = "/work/proj"
	tests := []struct {
		name        string
		args        []string
		left, right string
		wantErr     bool
	}{
		{name: "no args", args: nil, left: cwd, right: cwd},
		{name: "one relative", args: []string{"sub"}, left: "/work/proj/sub", right: "/work/proj/sub"},
		{name: "dot", args: []string{"."}, left: cwd, right: cwd},
		{name: "parent", args: []string{"../other"}, left: "/work/other", right: "/work/other"},
		{name: "two absolute", args: []string{"/a", "/b/"}, left: "/a", right: "/b"},
		{name: "mixed", args: []string{"/a", "x"}, left: "/a", right: "/work/proj/x"},
		{name: "too many", args: []string{"a", "b", "c"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, r, err := resolveDirs(tt.args, cwd)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if l != tt.left || r != tt.right {
				t.Errorf("got (%q, %q), want (%q, %q)", l, r, tt.left, tt.right)
			}
		})
	}
}

func TestRunExitCodes(t *testing.T) {
	errBoom := errors.New("terminal exploded")
	tests := []struct {
		name        string
		args        []string
		startErr    error
		wantCode    int
		wantStarted bool
		wantStderr  []string
	}{
		{name: "normal run", args: nil, wantCode: 0, wantStarted: true},
		{name: "two dirs", args: []string{"/a", "/b"}, wantCode: 0, wantStarted: true},
		{name: "too many args", args: []string{"a", "b", "c"}, wantCode: 1, wantStderr: []string{"fm:", "Usage:"}},
		{name: "unknown flag", args: []string{"-x"}, wantCode: 1, wantStderr: []string{"fm:", "-x", "Usage:"}},
		{name: "help", args: []string{"-h"}, wantCode: 0, wantStderr: []string{"Usage:"}},
		{name: "program error", args: nil, startErr: errBoom, wantCode: 1, wantStarted: true, wantStderr: []string{"fm: terminal exploded"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr bytes.Buffer
			started := false
			start := func(left, right string) error {
				started = true
				return tt.startErr
			}
			code := run(tt.args, "/cwd", &stderr, start)
			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d (stderr: %q)", code, tt.wantCode, stderr.String())
			}
			if started != tt.wantStarted {
				t.Errorf("started = %v, want %v", started, tt.wantStarted)
			}
			for _, s := range tt.wantStderr {
				if !strings.Contains(stderr.String(), s) {
					t.Errorf("stderr %q missing %q", stderr.String(), s)
				}
			}
			if len(tt.wantStderr) == 0 && stderr.Len() > 0 {
				t.Errorf("unexpected stderr: %q", stderr.String())
			}
		})
	}

	t.Run("passes resolved dirs to start", func(t *testing.T) {
		var gotL, gotR string
		run([]string{"sub"}, "/cwd", io.Discard, func(l, r string) error { gotL, gotR = l, r; return nil })
		if gotL != "/cwd/sub" || gotR != "/cwd/sub" {
			t.Errorf("start got (%q, %q)", gotL, gotR)
		}
	})
}
