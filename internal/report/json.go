package report

import (
	"encoding/json"
	"fmt"
	"io"
	"slices"

	"github.com/toprakgureli/sizinti/internal/scanner"
)

// JSON writes a redacted scan result as a single JSON document.
func JSON(writer io.Writer, result scanner.Result) error {
	result.Findings = slices.Clone(result.Findings)
	if result.Findings == nil {
		result.Findings = make([]scanner.Finding, 0)
	}
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("write JSON: %w", err)
	}
	return nil
}
