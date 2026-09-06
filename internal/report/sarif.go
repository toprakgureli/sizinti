package report

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"

	"github.com/toprakgureli/sizinti/internal/scanner"
)

type sarifLog struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name  string      `json:"name"`
	Rules []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID               string       `json:"id"`
	ShortDescription sarifMessage `json:"shortDescription"`
}

type sarifResult struct {
	RuleID     string          `json:"ruleId"`
	Level      string          `json:"level"`
	Message    sarifMessage    `json:"message"`
	Locations  []sarifLocation `json:"locations"`
	Properties sarifProperties `json:"properties"`
}

type sarifProperties struct {
	Confidence string `json:"confidence"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
	Region           sarifRegion           `json:"region"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

type sarifRegion struct {
	StartLine int `json:"startLine"`
}

// SARIF writes redacted SARIF 2.1.0 using scan-root-relative file URIs.
func SARIF(writer io.Writer, result scanner.Result) error {
	entries := make([]sarifResult, 0, len(result.Findings))
	definitions := make([]sarifRule, 0, len(result.Findings))
	seen := make(map[string]bool, len(result.Findings))
	for _, finding := range result.Findings {
		if !seen[finding.Rule] {
			seen[finding.Rule] = true
			definitions = append(definitions, sarifRule{
				ID: finding.Rule, ShortDescription: sarifMessage{Text: "Possible credential: " + finding.Rule},
			})
		}
		uri := url.URL{Path: finding.Path}
		entries = append(entries, sarifResult{
			RuleID: finding.Rule, Level: "warning",
			Message:    sarifMessage{Text: "Possible credential [REDACTED]; review and remove it."},
			Properties: sarifProperties{Confidence: finding.Confidence},
			Locations: []sarifLocation{{PhysicalLocation: sarifPhysicalLocation{
				ArtifactLocation: sarifArtifactLocation{URI: uri.String()},
				Region:           sarifRegion{StartLine: finding.Line},
			}}},
		})
	}
	document := sarifLog{
		Schema: "https://json.schemastore.org/sarif-2.1.0.json", Version: "2.1.0",
		Runs: []sarifRun{{Tool: sarifTool{Driver: sarifDriver{Name: "sizinti", Rules: definitions}}, Results: entries}},
	}
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(document); err != nil {
		return fmt.Errorf("write SARIF: %w", err)
	}
	return nil
}
