package scanner

import (
	"context"
	"errors"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/toprakgureli/sizinti/internal/allowlist"
	"github.com/toprakgureli/sizinti/internal/rules"
)

func newTestScanner(t testing.TB, options ...Option) *Scanner {
	t.Helper()
	set, err := rules.New()
	if err != nil {
		t.Fatal(err)
	}
	engine, err := New(set, options...)
	if err != nil {
		t.Fatal(err)
	}
	return engine
}

func writeTestFile(t testing.TB, root, name, content string) {
	t.Helper()
	filename := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestScanDetections(t *testing.T) {
	engine := newTestScanner(t)
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{name: "assignment", content: "port=8080\npassword=synthetic-only\n", want: "generic-assignment"},
		{name: "no final newline", content: "port=8080\npassword=synthetic-only", want: "generic-assignment"},
		{name: "CRLF", content: "port=8080\r\npassword=synthetic-only\r\n", want: "generic-assignment"},
		{name: "entropy", content: "port=8080\nAbCdEfGhIjKlMnOpQrStUvWxYz0123456789\n", want: "high-entropy"},
		{name: "safe", content: "port=8080\nname=demo\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeTestFile(t, root, "app.env", tt.content)
			result, err := engine.Scan(context.Background(), root)
			if err != nil {
				t.Fatal(err)
			}
			if result.Scanned != 1 {
				t.Fatalf("scanned = %d", result.Scanned)
			}
			if tt.want == "" {
				if len(result.Findings) != 0 {
					t.Fatal("safe content produced findings")
				}
				return
			}
			if len(result.Findings) != 1 {
				t.Fatalf("findings = %+v", result.Findings)
			}
			finding := result.Findings[0]
			if finding.Rule != tt.want || finding.Line != 2 || finding.Path != "app.env" || finding.Snippet != "[REDACTED]" {
				t.Fatalf("unexpected finding: %+v", finding)
			}
		})
	}
}

func TestSafeFixtureIsActuallyScanned(t *testing.T) {
	engine := newTestScanner(t)
	result, err := engine.Scan(context.Background(), filepath.Join("..", "..", "testdata", "safe"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Scanned != 1 || len(result.Findings) != 0 {
		t.Fatalf("safe fixture result = %+v", result)
	}
}

func TestSkipsAndFixtureOverride(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "fixtures/app.env", "password=synthetic-only")
	writeTestFile(t, root, "main_test.go", "password=synthetic-only")
	writeTestFile(t, root, "data.bin", "password=synthetic-only\n\x00")
	writeTestFile(t, root, "large.txt", strings.Repeat("a", 100))
	writeTestFile(t, root, "vendor/app.env", "password=synthetic-only")
	for _, include := range []bool{false, true} {
		settings := Defaults()
		settings.IncludeFixtures = include
		settings.MaxFileSize = 64
		engine := newTestScanner(t, WithSettings(settings))
		result, err := engine.Scan(context.Background(), root)
		if err != nil {
			t.Fatal(err)
		}
		want := 0
		if include {
			want = 2
		}
		if len(result.Findings) != want || result.Scanned != want || result.Skipped != 5-want {
			t.Fatalf("include=%v result=%+v", include, result)
		}
	}
}

func TestAllowlistAndNoEntropyDuplicates(t *testing.T) {
	value := "ghp_AbCdEfGhIjKlMnOpQrStUvWxYz0123456789"
	list, err := allowlist.Parse(strings.NewReader("regex:" + value + "\nglob:ignored.env\n"))
	if err != nil {
		t.Fatal(err)
	}
	engine := newTestScanner(t, WithAllowlist(list))
	root := t.TempDir()
	writeTestFile(t, root, "ignored.env", "password=another-synthetic")
	writeTestFile(t, root, "app.env", "api_key="+value+" password=another-synthetic")
	result, err := engine.Scan(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) != 1 || result.Findings[0].Rule != "generic-assignment" {
		t.Fatalf("unexpected findings: %+v", result.Findings)
	}
}

func TestDeterministicWorkers(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"z.env", "a.env", "nested/m.env"} {
		writeTestFile(t, root, name, "password=synthetic-only\niyzico_api_key=synthetic-only")
	}
	var previous Result
	for _, workers := range []int{1, 2, 8} {
		settings := Defaults()
		settings.Workers = workers
		engine := newTestScanner(t, WithSettings(settings))
		result, err := engine.Scan(context.Background(), root)
		if err != nil {
			t.Fatal(err)
		}
		if workers != 1 && !reflect.DeepEqual(result, previous) {
			t.Fatal("worker count changed output")
		}
		previous = result
	}
}

func TestScanFailures(t *testing.T) {
	engine := newTestScanner(t)
	t.Run("missing root", func(t *testing.T) {
		if _, err := engine.Scan(context.Background(), filepath.Join(t.TempDir(), "absent")); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("file root", func(t *testing.T) {
		root := t.TempDir()
		writeTestFile(t, root, "file", "safe")
		if _, err := engine.Scan(context.Background(), filepath.Join(root, "file")); err == nil {
			t.Fatal("file root accepted")
		}
	})
	t.Run("line limit", func(t *testing.T) {
		root := t.TempDir()
		writeTestFile(t, root, "long", strings.Repeat("a", maxLineSize+1))
		if _, err := engine.Scan(context.Background(), root); err == nil {
			t.Fatal("long line accepted")
		}
	})
	t.Run("cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := engine.Scan(ctx, t.TempDir()); !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestCancellationUnblocksResultSend(t *testing.T) {
	engine := newTestScanner(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events := make(chan event)
	done := make(chan error, 1)
	go func() {
		done <- engine.lines(ctx, "app.env", strings.NewReader("password=synthetic-only\npassword=another-synthetic"), events)
	}()
	select {
	case <-events:
	case <-time.After(5 * time.Second):
		t.Fatal("no initial finding")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("worker did not exit")
	}
}

func TestSettingsValidation(t *testing.T) {
	for _, settings := range []Settings{
		{Workers: 0, MaxFileSize: 1},
		{Workers: 1025, MaxFileSize: 1},
		{Workers: 1, MaxFileSize: 0},
		{Workers: 1, MaxFileSize: 1, EntropyThreshold: math.NaN()},
		{Workers: 1, MaxFileSize: 1, EntropyThreshold: math.Inf(1)},
		{Workers: 1, MaxFileSize: 1, EntropyThreshold: 9},
	} {
		if err := settings.Validate(); err == nil {
			t.Errorf("accepted %+v", settings)
		}
	}
	if _, err := New(nil); err == nil {
		t.Fatal("nil rules accepted")
	}
	set, err := rules.New()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := New(set, nil); err == nil {
		t.Fatal("nil option accepted")
	}
}

func TestEntropyThreshold(t *testing.T) {
	settings := Defaults()
	settings.EntropyThreshold = 8
	engine := newTestScanner(t, WithSettings(settings))
	if matches := engine.detect("AbCdEfGhIjKlMnOpQrStUvWxYz0123456789"); len(matches) != 0 {
		t.Fatalf("high threshold produced %d findings", len(matches))
	}
}
