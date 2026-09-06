package config

import (
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		invalid bool
	}{
		{name: "empty"},
		{name: "override", yaml: "workers: 2\nformat: json\n"},
		{name: "unknown field", yaml: "worker: 2", invalid: true},
		{name: "wrong type", yaml: "workers: secret-value", invalid: true},
		{name: "multiple documents", yaml: "workers: 2\n---\nworkers: 3", invalid: true},
		{name: "duplicate key", yaml: "workers: 2\nworkers: 3", invalid: true},
		{name: "too large", yaml: strings.Repeat(" ", (1<<20)+1), invalid: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings, err := Load(strings.NewReader(tt.yaml))
			if (err != nil) != tt.invalid {
				t.Fatalf("error = %v", err)
			}
			if err != nil {
				if strings.Contains(err.Error(), "secret-value") {
					t.Fatal("configuration value leaked")
				}
				return
			}
			if settings.MaxFileSize != Defaults().MaxFileSize {
				t.Fatal("omitted default was lost")
			}
			if tt.name == "override" && (settings.Workers != 2 || settings.Format != "json") {
				t.Fatal("overrides were lost")
			}
		})
	}
}

func TestValidate(t *testing.T) {
	for _, format := range []string{"table", "json", "sarif", "invalid"} {
		settings := Defaults()
		settings.Format = format
		if (settings.Validate() != nil) != (format == "invalid") {
			t.Errorf("unexpected validation for %s", format)
		}
	}
	settings := Defaults()
	settings.Workers = 0
	if settings.Validate() == nil {
		t.Fatal("zero workers accepted")
	}
}
