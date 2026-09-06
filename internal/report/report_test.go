package report

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/toprakgureli/sizinti/internal/scanner"
)

type failingWriter struct {
	err error
}

var _ io.Writer = (*failingWriter)(nil)

func (w *failingWriter) Write([]byte) (int, error) { return 0, w.err }

func TestReports(t *testing.T) {
	result := scanner.Result{Findings: []scanner.Finding{{
		Path: "config/a #b.env", Line: 7, Rule: "iyzico", Snippet: "[REDACTED]", Confidence: "medium",
	}}, Scanned: 1}
	tests := []struct {
		name  string
		write func(io.Writer, scanner.Result) error
	}{
		{name: "json", write: JSON},
		{name: "sarif", write: SARIF},
		{name: "table", write: func(w io.Writer, r scanner.Result) error { return Table(w, r, false) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			if err := tt.write(&output, result); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(output.String(), "[REDACTED]") || !strings.Contains(output.String(), "iyzico") {
				t.Fatal("report lacks redacted finding")
			}
			failure := errors.New("sink failed")
			if err := tt.write(&failingWriter{err: failure}, result); !errors.Is(err, failure) {
				t.Fatalf("writer error was lost: %v", err)
			}
		})
	}
	var output bytes.Buffer
	if err := SARIF(&output, result); err != nil {
		t.Fatal(err)
	}
	var document sarifLog
	if err := json.Unmarshal(output.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	if document.Version != "2.1.0" || len(document.Runs) != 1 || document.Runs[0].Tool.Driver.Name != "sizinti" {
		t.Fatal("invalid SARIF envelope")
	}
	entry := document.Runs[0].Results[0]
	location := entry.Locations[0].PhysicalLocation
	if location.ArtifactLocation.URI != "config/a%20%23b.env" || location.Region.StartLine != 7 || entry.RuleID != "iyzico" {
		t.Fatalf("incorrect SARIF location: %+v", location)
	}
}

func TestEmptyReports(t *testing.T) {
	for _, writer := range []func(io.Writer, scanner.Result) error{JSON, SARIF} {
		var output bytes.Buffer
		if err := writer(&output, scanner.Result{}); err != nil {
			t.Fatal(err)
		}
		if !json.Valid(output.Bytes()) || strings.Contains(output.String(), "null") {
			t.Fatalf("invalid empty document: %s", output.String())
		}
	}
}

func TestTableEscapesPaths(t *testing.T) {
	var output bytes.Buffer
	result := scanner.Result{Findings: []scanner.Finding{{Path: "evil\x1b[2J\n.env", Line: 1, Snippet: "[REDACTED]"}}}
	if err := Table(&output, result, false); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "\x1b") || !strings.Contains(output.String(), `\x1b`) {
		t.Fatal("terminal escape was not quoted")
	}
	output.Reset()
	if err := Table(&output, scanner.Result{}, true); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "\x1b[1;31m") {
		t.Fatal("color option was lost")
	}
}
