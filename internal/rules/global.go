package rules

func global() []definition {
	return []definition{
		{name: "aws-access-key", pattern: `\b(?:AKIA|ASIA)[A-Z0-9]{16}\b`, confidence: "high"},
		{name: "aws-secret-key", pattern: `(?i)\baws[_-]?secret[_-]?access[_-]?key["']?\s*[:=]\s*["']?([A-Za-z0-9/+=]{40})(?:[^A-Za-z0-9/+=]|$)`, confidence: "high", group: 1},
		{name: "gcp-service-account", pattern: `"type"\s*:\s*"(service_account)"`, confidence: "low", group: 1},
		{name: "github-token", pattern: `\bgh[po]_[A-Za-z0-9]{36}\b`, confidence: "high"},
		{name: "slack-token", pattern: `\bxox[baprs]-[A-Za-z0-9-]{10,}\b`, confidence: "high"},
		{name: "stripe-key", pattern: `\bsk_(?:live|test)_[A-Za-z0-9]{16,}\b`, confidence: "high"},
		{name: "jwt", pattern: `\beyJ[A-Za-z0-9_-]{5,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}`, confidence: "medium"},
		{name: "pem-private-key", pattern: `-----BEGIN (?:RSA |EC |DSA |OPENSSH |ENCRYPTED )?PRIVATE KEY-----`, confidence: "high"},
		{name: "database-url", pattern: `(?i)\b(?:postgres(?:ql)?|mysql|mongodb(?:\+srv)?)://[^\s:"'/@]+:([^\s@"']+)@`, confidence: "high", group: 1},
	}
}

func generic() definition {
	return definition{
		name:       "generic-assignment",
		pattern:    `(?i)\b(?:api[_-]?key|secret|password|passwd|token|client[_-]?secret)["']?\s*[:=]\s*["']?([^\s"'\x60;,}{]{4,})`,
		confidence: "medium", group: 1,
	}
}
