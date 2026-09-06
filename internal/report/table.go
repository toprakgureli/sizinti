package report

import (
	"fmt"
	"io"
	"strconv"
	"text/tabwriter"

	"github.com/toprakgureli/sizinti/internal/scanner"
)

// Table writes a readable table with optional ANSI heading color.
func Table(writer io.Writer, result scanner.Result, color bool) error {
	table := tabwriter.NewWriter(writer, 0, 4, 2, ' ', 0)
	heading := "FILE\tLINE\tRULE\tCONFIDENCE\tSNIPPET"
	if color {
		heading = "\x1b[1;31m" + heading + "\x1b[0m"
	}
	if _, err := fmt.Fprintln(table, heading); err != nil {
		return fmt.Errorf("write table heading: %w", err)
	}
	for _, finding := range result.Findings {
		if _, err := fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\n",
			strconv.QuoteToGraphic(finding.Path), strconv.Itoa(finding.Line),
			finding.Rule, finding.Confidence, finding.Snippet); err != nil {
			return fmt.Errorf("write table row: %w", err)
		}
	}
	if err := table.Flush(); err != nil {
		return fmt.Errorf("flush table: %w", err)
	}
	if _, err := fmt.Fprintf(writer, "%d finding(s); %d scanned; %d skipped entries\n",
		len(result.Findings), result.Scanned, result.Skipped); err != nil {
		return fmt.Errorf("write table summary: %w", err)
	}
	return nil
}
