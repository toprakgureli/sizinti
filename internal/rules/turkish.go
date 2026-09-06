package rules

func turkish() []definition {
	providers := []struct {
		name   string
		fields string
	}{
		{name: "iyzico", fields: `iyzico[_-](?:api[_-]?key|secret[_-]?key)`},
		{name: "paytr", fields: `paytr[_-](?:merchant[_-]?(?:key|salt)|api[_-]?key)`},
		{name: "param", fields: `param[_-](?:client[_-]?(?:code|username|password)|guid|api[_-]?key)`},
		{name: "netgsm", fields: `netgsm[_-](?:password|usercode|api[_-]?key)`},
		{name: "turkish-bank-api", fields: `(?:banka|bank|turkish[_-]bank|tr[_-]bank)[_-](?:api[_-]?key|client[_-]?secret|password)`},
	}
	definitions := make([]definition, 0, len(providers))
	for _, provider := range providers {
		definitions = append(definitions, definition{
			name:       provider.name,
			pattern:    `(?i)\b` + provider.fields + `["']?\s*[:=]\s*["']?([^\s"'\x60;,}{]{4,})`,
			confidence: "medium", group: 1,
		})
	}
	return definitions
}
