package scanner

// Finding is a redacted detection safe to pass to report writers.
type Finding struct {
	Path       string `json:"path"`
	Line       int    `json:"line"`
	Rule       string `json:"rule"`
	Snippet    string `json:"snippet"`
	Confidence string `json:"confidence"`
}

// Result contains sorted findings and scan counts.
type Result struct {
	Findings []Finding `json:"findings"`
	Scanned  int       `json:"scanned"`
	Skipped  int       `json:"skipped"`
}
