package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"

	"github.com/spf13/cobra"

	"github.com/toprakgureli/sizinti/internal/allowlist"
	"github.com/toprakgureli/sizinti/internal/config"
	"github.com/toprakgureli/sizinti/internal/report"
	"github.com/toprakgureli/sizinti/internal/rules"
	"github.com/toprakgureli/sizinti/internal/scanner"
)

type findingsError struct{}

var _ error = (*findingsError)(nil)

func (*findingsError) Error() string { return "possible credentials found" }

// Run executes the CLI and returns an exit code without terminating the process.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	command := newRoot(stdout, stderr)
	command.SetArgs(slices.Clone(args))
	err := command.ExecuteContext(ctx)
	if err == nil {
		return 0
	}
	var findingErr *findingsError
	if errors.As(err, &findingErr) {
		return 1
	}
	if _, writeErr := fmt.Fprintf(stderr, "sizinti: %s\n", err); writeErr != nil {
		return 2
	}
	return 2
}

func newRoot(stdout, stderr io.Writer) *cobra.Command {
	flags := config.Defaults()
	var configPath string
	var jsonOutput, sarifOutput bool
	command := &cobra.Command{
		Use: "sizinti [directory]", Short: "Find possible credentials before they reach a commit",
		Args: cobra.MaximumNArgs(1), SilenceUsage: true, SilenceErrors: true,
		RunE: func(command *cobra.Command, args []string) error {
			root := "."
			if len(args) > 0 {
				root = args[0]
			}
			settings, err := loadConfig(configPath)
			if err != nil {
				return err
			}
			applyFlags(command, flags, &settings)
			if jsonOutput {
				settings.Format = "json"
			}
			if sarifOutput {
				settings.Format = "sarif"
			}
			if err := settings.Validate(); err != nil {
				return err
			}
			return scan(command.Context(), root, settings, command.OutOrStdout())
		},
	}
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.CompletionOptions.DisableDefaultCmd = true
	options := command.Flags()
	options.StringVar(&configPath, "config", "", "Optional YAML configuration file")
	options.IntVar(&flags.Workers, "workers", flags.Workers, "Number of file workers (1-1024)")
	options.Float64Var(&flags.EntropyThreshold, "entropy-threshold", flags.EntropyThreshold, "Entropy threshold in bits per byte (0-8)")
	options.Int64Var(&flags.MaxFileSize, "max-file-size", flags.MaxFileSize, "Maximum file size in bytes")
	options.BoolVar(&flags.IncludeFixtures, "include-fixtures", false, "Include common test and example paths")
	options.StringVar(&flags.IgnoreFile, "ignore-file", flags.IgnoreFile, "Ignore file relative to scan root; empty disables it")
	options.BoolVar(&flags.NoColor, "no-color", false, "Disable ANSI table colors")
	options.BoolVar(&jsonOutput, "json", false, "Write JSON")
	options.BoolVar(&sarifOutput, "sarif", false, "Write SARIF 2.1.0")
	command.MarkFlagsMutuallyExclusive("json", "sarif")
	return command
}

func applyFlags(command *cobra.Command, flags config.Config, settings *config.Config) {
	if command.Flags().Changed("workers") {
		settings.Workers = flags.Workers
	}
	if command.Flags().Changed("entropy-threshold") {
		settings.EntropyThreshold = flags.EntropyThreshold
	}
	if command.Flags().Changed("max-file-size") {
		settings.MaxFileSize = flags.MaxFileSize
	}
	if command.Flags().Changed("include-fixtures") {
		settings.IncludeFixtures = flags.IncludeFixtures
	}
	if command.Flags().Changed("ignore-file") {
		settings.IgnoreFile = flags.IgnoreFile
	}
	if command.Flags().Changed("no-color") {
		settings.NoColor = flags.NoColor
	}
}

func loadConfig(name string) (settings config.Config, err error) {
	if name == "" {
		return config.Defaults(), nil
	}
	file, err := os.Open(name)
	if err != nil {
		return settings, fmt.Errorf("open configuration: %w", err)
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	return config.Load(file)
}

func loadAllowlist(root, name string) (list *allowlist.List, err error) {
	if name == "" {
		return nil, nil
	}
	filename := name
	if !filepath.IsAbs(filename) {
		filename = filepath.Join(root, filename)
	}
	file, err := os.Open(filename)
	if errors.Is(err, os.ErrNotExist) && name == ".sizintiignore" {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open ignore file: %w", err)
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	return allowlist.Parse(file)
}

func scan(ctx context.Context, root string, settings config.Config, writer io.Writer) error {
	list, err := loadAllowlist(root, settings.IgnoreFile)
	if err != nil {
		return err
	}
	set, err := rules.New()
	if err != nil {
		return err
	}
	engine, err := scanner.New(set, scanner.WithSettings(settings.Settings()), scanner.WithAllowlist(list))
	if err != nil {
		return err
	}
	result, err := engine.Scan(ctx, root)
	if err != nil {
		return err
	}
	switch settings.Format {
	case "json":
		err = report.JSON(writer, result)
	case "sarif":
		err = report.SARIF(writer, result)
	default:
		err = report.Table(writer, result, !settings.NoColor && os.Getenv("NO_COLOR") == "" && terminal(writer))
	}
	if err != nil {
		return err
	}
	if len(result.Findings) > 0 {
		return &findingsError{}
	}
	return nil
}

func terminal(writer io.Writer) bool {
	file, ok := writer.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
