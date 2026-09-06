package scanner

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/toprakgureli/sizinti/internal/allowlist"
	"github.com/toprakgureli/sizinti/internal/rules"
)

// Settings controls resource limits and detection sensitivity.
type Settings struct {
	Workers          int
	EntropyThreshold float64
	MaxFileSize      int64
	IncludeFixtures  bool
}

// Defaults returns the default scanner settings.
func Defaults() Settings {
	return Settings{Workers: runtime.NumCPU(), EntropyThreshold: 4.5, MaxFileSize: 10 << 20}
}

// Validate checks scanner settings before any goroutine starts.
func (s Settings) Validate() error {
	if s.Workers < 1 || s.Workers > 1024 {
		return errors.New("workers must be between 1 and 1024")
	}
	if math.IsNaN(s.EntropyThreshold) || math.IsInf(s.EntropyThreshold, 0) || s.EntropyThreshold < 0 || s.EntropyThreshold > 8 {
		return errors.New("entropy threshold must be finite and between 0 and 8")
	}
	if s.MaxFileSize < 1 || s.MaxFileSize == math.MaxInt64 {
		return errors.New("max file size must be positive and less than MaxInt64")
	}
	return nil
}

// Option configures a scanner at construction time.
type Option func(*Scanner)

// WithSettings replaces the default scalar settings.
func WithSettings(settings Settings) Option {
	return func(s *Scanner) { s.settings = settings }
}

// WithAllowlist supplies an immutable parsed allowlist.
func WithAllowlist(list *allowlist.List) Option {
	return func(s *Scanner) { s.allowlist = list }
}

// Scanner scans regular files with a bounded worker pool.
type Scanner struct {
	settings   Settings
	rules      *rules.Set
	allowlist  *allowlist.List
	candidates *regexp.Regexp
}

// New constructs a scanner with immutable detection rules.
func New(set *rules.Set, options ...Option) (*Scanner, error) {
	if set == nil {
		return nil, errors.New("rules must not be nil")
	}
	s := &Scanner{settings: Defaults(), rules: set}
	for _, option := range options {
		if option == nil {
			return nil, errors.New("scanner option must not be nil")
		}
		option(s)
	}
	if err := s.settings.Validate(); err != nil {
		return nil, err
	}
	candidates, err := regexp.Compile(`[A-Za-z0-9_+/=-]{20,}`)
	if err != nil {
		return nil, fmt.Errorf("compile entropy candidates: %w", err)
	}
	s.candidates = candidates
	return s, nil
}

type event struct {
	finding *Finding
	scanned int
	skipped int
}

// Scan scans a directory and joins all goroutines before returning.
func (s *Scanner) Scan(parent context.Context, root string) (Result, error) {
	var result Result
	info, err := os.Lstat(root)
	if err != nil {
		return result, fmt.Errorf("inspect scan root: %w", err)
	}
	if !info.IsDir() {
		return result, errors.New("scan root must be a directory, not a symlink or file")
	}
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	paths := make(chan string)
	events := make(chan event)
	failures := make(chan error, 1)
	fail := func(err error) {
		select {
		case failures <- err:
		default:
		}
		cancel()
	}
	var group sync.WaitGroup
	group.Add(1)
	go func() {
		defer group.Done()
		defer close(paths)
		if err := s.walk(ctx, root, paths, events); err != nil {
			fail(err)
		}
	}()
	for i := 0; i < s.settings.Workers; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for name := range paths {
				if err := s.file(ctx, root, name, events); err != nil {
					fail(err)
					return
				}
			}
		}()
	}
	go func() {
		group.Wait()
		close(events)
	}()
	for item := range events {
		if item.finding != nil {
			result.Findings = append(result.Findings, *item.finding)
		}
		result.Scanned += item.scanned
		result.Skipped += item.skipped
	}
	sort.Slice(result.Findings, func(i, j int) bool {
		a, b := result.Findings[i], result.Findings[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Rule < b.Rule
	})
	select {
	case err := <-failures:
		return result, err
	default:
		return result, parent.Err()
	}
}

func (s *Scanner) walk(ctx context.Context, root string, paths chan<- string, events chan<- event) error {
	return filepath.WalkDir(root, func(name string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return fmt.Errorf("walk %q: %w", name, walkErr)
		}
		if name == root {
			return nil
		}
		relative, err := filepath.Rel(root, name)
		if err != nil {
			return fmt.Errorf("relative path: %w", err)
		}
		relative = filepath.ToSlash(relative)
		if s.ignored(relative, entry.IsDir()) || !entry.IsDir() && !entry.Type().IsRegular() {
			if err := send(ctx, events, event{skipped: 1}); err != nil {
				return err
			}
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		select {
		case paths <- name:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
}

func (s *Scanner) ignored(name string, directory bool) bool {
	if s.allowlist.Path(name) {
		return true
	}
	base := filepath.Base(name)
	if directory && (base == ".git" || base == "node_modules" || base == "vendor") {
		return true
	}
	if s.settings.IncludeFixtures {
		return false
	}
	if directory {
		return base == "testdata" || base == "fixtures" || base == "examples"
	}
	return strings.HasSuffix(base, "_test.go") || strings.HasSuffix(base, ".example") || strings.HasSuffix(base, ".sample")
}

func send(ctx context.Context, events chan<- event, item event) error {
	select {
	case events <- item:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
