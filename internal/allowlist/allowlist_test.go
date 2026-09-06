package allowlist

import (
	"strings"
	"testing"
)

func TestMatches(t *testing.T) {
	list, err := Parse(strings.NewReader("# local exceptions\nglob:*.sample\nglob:config/*.local\nregex:synthetic-[a-z]+\nregex:demo|demo-long\n"))
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name  string
		value string
		path  bool
		want  bool
	}{
		{name: "basename", value: "deep/demo.sample", path: true, want: true},
		{name: "root glob", value: "config/dev.local", path: true, want: true},
		{name: "anchored path", value: "nested/config/dev.local", path: true},
		{name: "ordinary file", value: "main.go", path: true},
		{name: "whole value", value: "synthetic-only", want: true},
		{name: "whole alternative", value: "demo-long", want: true},
		{name: "partial value", value: "prefix-synthetic-only"},
		{name: "different value", value: "another-secret"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := list.Value(tt.value)
			if tt.path {
				got = list.Path(tt.value)
			}
			if got != tt.want {
				t.Errorf("match = %v, want %v", got, tt.want)
			}
		})
	}
	var empty *List
	if empty.Path("file") || empty.Value("value") {
		t.Fatal("nil list must not ignore anything")
	}
}

func TestInvalidEntries(t *testing.T) {
	for _, input := range []string{"missing-prefix", "glob:", "regex:[", "glob:[", "other:value"} {
		t.Run(input, func(t *testing.T) {
			if _, err := Parse(strings.NewReader(input)); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
