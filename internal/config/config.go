package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"

	"github.com/toprakgureli/sizinti/internal/scanner"
)

// Config defines YAML and CLI settings.
type Config struct {
	Workers          int     `yaml:"workers"`
	EntropyThreshold float64 `yaml:"entropy-threshold"`
	MaxFileSize      int64   `yaml:"max-file-size"`
	IncludeFixtures  bool    `yaml:"include-fixtures"`
	IgnoreFile       string  `yaml:"ignore-file"`
	Format           string  `yaml:"format"`
	NoColor          bool    `yaml:"no-color"`
}

// Defaults returns a fresh configuration with conservative resource limits.
func Defaults() Config {
	settings := scanner.Defaults()
	return Config{
		Workers: settings.Workers, EntropyThreshold: settings.EntropyThreshold,
		MaxFileSize: settings.MaxFileSize, IgnoreFile: ".sizintiignore", Format: "table",
	}
}

// Load decodes one strict YAML document over the defaults.
func Load(reader io.Reader) (Config, error) {
	data, err := io.ReadAll(io.LimitReader(reader, (1<<20)+1))
	if err != nil {
		return Config{}, fmt.Errorf("read configuration: %w", err)
	}
	if len(data) > 1<<20 {
		return Config{}, errors.New("configuration exceeds 1 MiB")
	}
	config := Defaults()
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&config); err != nil && !errors.Is(err, io.EOF) {
		return Config{}, errors.New("invalid YAML configuration: check field names and value types")
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return Config{}, errors.New("configuration must contain exactly one YAML document")
	}
	return config, nil
}

// Settings returns the scanner's scalar settings by value.
func (c Config) Settings() scanner.Settings {
	return scanner.Settings{
		Workers: c.Workers, EntropyThreshold: c.EntropyThreshold,
		MaxFileSize: c.MaxFileSize, IncludeFixtures: c.IncludeFixtures,
	}
}

// Validate rejects invalid effective settings after flag overrides.
func (c Config) Validate() error {
	if err := c.Settings().Validate(); err != nil {
		return fmt.Errorf("configuration: %w", err)
	}
	switch c.Format {
	case "table", "json", "sarif":
		return nil
	default:
		return errors.New("format must be table, json, or sarif")
	}
}
