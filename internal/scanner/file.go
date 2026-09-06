package scanner

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/toprakgureli/sizinti/internal/entropy"
	"github.com/toprakgureli/sizinti/internal/rules"
)

const maxLineSize = 1 << 20

func (s *Scanner) file(ctx context.Context, root, name string, events chan<- event) (err error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	info, err := os.Lstat(name)
	if err != nil {
		return fmt.Errorf("inspect %q: %w", name, err)
	}
	if !info.Mode().IsRegular() || info.Size() > s.settings.MaxFileSize {
		return send(ctx, events, event{skipped: 1})
	}
	file, err := os.Open(name)
	if err != nil {
		return fmt.Errorf("open %q: %w", name, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close %q: %w", name, closeErr))
		}
	}()
	opened, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat %q: %w", name, err)
	}
	if !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		return fmt.Errorf("file changed while opening %q", name)
	}
	binary, err := binaryFile(ctx, io.LimitReader(file, s.settings.MaxFileSize+1), s.settings.MaxFileSize)
	if err != nil {
		return fmt.Errorf("classify %q: %w", name, err)
	}
	if binary {
		return send(ctx, events, event{skipped: 1})
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("rewind %q: %w", name, err)
	}
	relative, err := filepath.Rel(root, name)
	if err != nil {
		return fmt.Errorf("relative path: %w", err)
	}
	limited := &io.LimitedReader{R: file, N: s.settings.MaxFileSize + 1}
	if err := s.lines(ctx, filepath.ToSlash(relative), limited, events); err != nil {
		return fmt.Errorf("scan %q: %w", name, err)
	}
	if limited.N == 0 {
		return fmt.Errorf("file grew beyond max file size: %q", name)
	}
	return send(ctx, events, event{scanned: 1})
}

func binaryFile(ctx context.Context, reader io.Reader, maximum int64) (bool, error) {
	buffer := make([]byte, 32<<10)
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		n, err := reader.Read(buffer)
		total += int64(n)
		if total > maximum {
			return false, errors.New("file grew beyond max file size")
		}
		for _, value := range buffer[:n] {
			if value < 9 || value > 13 && value < 32 {
				return true, nil
			}
		}
		if errors.Is(err, io.EOF) {
			return false, nil
		}
		if err != nil {
			return false, fmt.Errorf("read content: %w", err)
		}
	}
}

func (s *Scanner) lines(ctx context.Context, name string, reader io.Reader, events chan<- event) error {
	lines := bufio.NewScanner(reader)
	lines.Buffer(make([]byte, 64<<10), maxLineSize)
	lineNumber := 0
	for lines.Scan() {
		if err := ctx.Err(); err != nil {
			return err
		}
		lineNumber++
		line := lines.Text()
		for _, match := range s.detect(line) {
			finding := Finding{
				Path: name, Line: lineNumber, Rule: match.Rule,
				Snippet: "[REDACTED]", Confidence: match.Confidence,
			}
			if err := send(ctx, events, event{finding: &finding}); err != nil {
				return err
			}
		}
	}
	if err := lines.Err(); err != nil {
		return fmt.Errorf("read lines (maximum line buffer is 1 MiB): %w", err)
	}
	return nil
}

func (s *Scanner) detect(line string) []rules.Match {
	known := s.rules.Find(line)
	matches := make([]rules.Match, 0, len(known))
	for _, match := range known {
		if !s.allowlist.Value(line[match.Start:match.End]) {
			matches = append(matches, match)
		}
	}
	for _, indices := range s.candidates.FindAllStringIndex(line, -1) {
		if covered(known, indices[0], indices[1]) {
			continue
		}
		value := line[indices[0]:indices[1]]
		if s.allowlist.Value(value) || entropy.Shannon(value) < s.settings.EntropyThreshold {
			continue
		}
		matches = append(matches, rules.Match{
			Start: indices[0], End: indices[1], Rule: "high-entropy", Confidence: "low",
		})
	}
	return matches
}

func covered(matches []rules.Match, start, end int) bool {
	for _, match := range matches {
		if start < match.End && end > match.Start {
			return true
		}
	}
	return false
}
