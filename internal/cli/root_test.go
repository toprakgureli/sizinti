package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, name, content string) {
	t.Helper()
	if err := os.WriteFile(name, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestExitCodesAndRedaction(t *testing.T) {
	tests := []struct {
		name    string
		content string
		args    []string
		want    int
	}{
		{name: "clean", content: "PORT=8080", args: []string{"--json"}},
		{name: "table finding", content: "password=synthetic-only", want: 1},
		{name: "JSON finding", content: "password=synthetic-only", args: []string{"--json"}, want: 1},
		{name: "SARIF finding", content: "password=synthetic-only", args: []string{"--sarif"}, want: 1},
		{name: "invalid workers", args: []string{"--workers=0"}, want: 2},
		{name: "invalid threshold", args: []string{"--entropy-threshold=NaN"}, want: 2},
		{name: "conflicting output", args: []string{"--json", "--sarif"}, want: 2},
		{name: "help", args: []string{"--help"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, filepath.Join(root, "app.env"), tt.content)
			args := append([]string{root}, tt.args...)
			var stdout, stderr bytes.Buffer
			if got := Run(context.Background(), args, &stdout, &stderr); got != tt.want {
				t.Fatalf("code = %d, want %d; stderr=%s", got, tt.want, stderr.String())
			}
			if strings.Contains(stdout.String()+stderr.String(), "synthetic-only") {
				t.Fatal("secret leaked through CLI")
			}
			if tt.want == 2 && (stdout.Len() != 0 || stderr.Len() == 0) {
				t.Fatal("error stream separation failed")
			}
			if tt.want != 2 && len(tt.args) > 0 && (tt.args[0] == "--json" || tt.args[0] == "--sarif") {
				if !json.Valid(stdout.Bytes()) || stderr.Len() != 0 {
					t.Fatal("machine output is not a standalone JSON document")
				}
			}
		})
	}
}

func TestConfigPrecedence(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, configPath, "workers: 0\nformat: sarif\ninclude-fixtures: true\n")
	writeFile(t, filepath.Join(root, "app_test.go"), "password=synthetic-only")
	var stdout, stderr bytes.Buffer
	args := []string{root, "--config", configPath, "--workers=2", "--include-fixtures=false", "--json"}
	if code := Run(context.Background(), args, &stdout, &stderr); code != 0 {
		t.Fatalf("code = %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"findings": []`) || strings.Contains(stdout.String(), `"runs"`) {
		t.Fatal("flags did not override config")
	}
}

func TestIgnoreFileAndMissingInputs(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "app.env"), "password=synthetic-only")
	writeFile(t, filepath.Join(root, ".sizintiignore"), "glob:app.env\n")
	for _, tt := range []struct {
		name string
		args []string
		want int
	}{
		{name: "default ignore", args: []string{root}},
		{name: "disabled ignore", args: []string{root, "--ignore-file="}, want: 1},
		{name: "missing custom ignore", args: []string{root, "--ignore-file=absent"}, want: 2},
		{name: "missing config", args: []string{root, "--config=absent"}, want: 2},
		{name: "missing root", args: []string{filepath.Join(root, "absent")}, want: 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := Run(context.Background(), tt.args, &stdout, &stderr); code != tt.want {
				t.Fatalf("code = %d, stderr=%s", code, stderr.String())
			}
		})
	}
}
