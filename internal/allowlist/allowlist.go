package allowlist

import (
	"bufio"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"
)

// List holds immutable path globs and candidate exceptions.
type List struct {
	paths       []string
	expressions []*regexp.Regexp
}

// Parse reads glob: and regex: entries without exposing invalid expressions.
func Parse(reader io.Reader) (*List, error) {
	list := &List{}
	lines := bufio.NewScanner(reader)
	lineNumber := 0
	for lines.Scan() {
		lineNumber++
		line := strings.TrimSpace(lines.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		kind, value, found := strings.Cut(line, ":")
		value = strings.TrimSpace(value)
		if !found || value == "" {
			return nil, fmt.Errorf("invalid ignore entry at line %d", lineNumber)
		}
		switch kind {
		case "glob":
			if _, err := path.Match(value, ""); err != nil {
				return nil, fmt.Errorf("invalid ignore glob at line %d", lineNumber)
			}
			list.paths = append(list.paths, value)
		case "regex":
			expression, err := regexp.Compile(`\A(?:` + value + `)\z`)
			if err != nil {
				return nil, fmt.Errorf("invalid ignore regex at line %d", lineNumber)
			}
			list.expressions = append(list.expressions, expression)
		default:
			return nil, fmt.Errorf("unknown ignore entry at line %d", lineNumber)
		}
	}
	if err := lines.Err(); err != nil {
		return nil, fmt.Errorf("read ignore file: %w", err)
	}
	return list, nil
}

// Path reports whether a root-relative slash path is ignored.
func (l *List) Path(name string) bool {
	if l == nil {
		return false
	}
	for _, pattern := range l.paths {
		candidate := name
		if !strings.Contains(pattern, "/") {
			candidate = path.Base(name)
		}
		matched, _ := path.Match(pattern, candidate)
		if matched {
			return true
		}
	}
	return false
}

// Value reports whether an entire candidate matches an exception.
func (l *List) Value(value string) bool {
	if l == nil {
		return false
	}
	for _, expression := range l.expressions {
		if expression.MatchString(value) {
			return true
		}
	}
	return false
}
