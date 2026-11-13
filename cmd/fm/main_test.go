package main

import "testing"

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
