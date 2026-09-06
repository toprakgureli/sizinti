package rules

import (
	"fmt"
	"regexp"
)

// Match identifies a credential span without retaining its value.
type Match struct {
	Start      int
	End        int
	Rule       string
	Confidence string
}

type definition struct {
	name       string
	pattern    string
	confidence string
	group      int
}

type rule struct {
	definition definition
	expression *regexp.Regexp
}

// Set contains immutable compiled detection rules.
type Set struct {
	rules []rule
}

// New compiles the built-in global and Turkish rules.
func New() (*Set, error) {
	definitions := append(global(), turkish()...)
	definitions = append(definitions, generic())
	compiled := make([]rule, 0, len(definitions))
	for _, item := range definitions {
		expression, err := regexp.Compile(item.pattern)
		if err != nil {
			return nil, fmt.Errorf("compile rule %s: %w", item.name, err)
		}
		compiled = append(compiled, rule{definition: item, expression: expression})
	}
	return &Set{rules: compiled}, nil
}

// Find returns non-overlapping credential spans in rule priority order.
func (s *Set) Find(line string) []Match {
	var matches []Match
	for _, item := range s.rules {
		for _, indices := range item.expression.FindAllStringSubmatchIndex(line, -1) {
			start, end := indices[2*item.definition.group], indices[2*item.definition.group+1]
			if start < 0 || overlaps(matches, start, end) {
				continue
			}
			matches = append(matches, Match{
				Start: start, End: end, Rule: item.definition.name,
				Confidence: item.definition.confidence,
			})
		}
	}
	return matches
}

func overlaps(matches []Match, start, end int) bool {
	for _, match := range matches {
		if start < match.End && end > match.Start {
			return true
		}
	}
	return false
}
