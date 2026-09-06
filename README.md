🇬🇧 English | 🇹🇷 [Türkçe](README.tr.md)

# sizinti

[![CI workflow](https://img.shields.io/badge/CI-workflow%20included-blue)](.github/workflows/ci.yml)
![Go](https://img.shields.io/badge/Go-1.25%2B-00ADD8)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Concurrent secret scanner in Go — catches leaked credentials before they reach a commit, with Turkish provider rules.

## Why I built this

As a backend engineer and former penetration tester, I care about catching credentials while they are still on a developer's machine. I built sizinti to explore that problem with readable Go code I can explain, and to give Turkish provider configurations explicit attention.

Sizinti reads local files and reports possible credentials. It never uses discovered values to contact a provider. The CI badge links to the included workflow; replace it with a live status badge after publishing and running CI.

## When to use it

- **Before a commit:** run `sizinti .` locally or from an existing pre-commit hook.
- **On a push or pull request:** use it as a required CI check to block detected credentials.
- **During an audit:** scan an existing working tree, including untracked files and, when needed, fixtures.

The current scan covers files on disk. A staged version can differ from the working tree; staged-index and Git history scanning are on the roadmap.

## Features

- Bounded worker pool, context cancellation, and deterministic result ordering.
- Regex rules plus Shannon entropy detection for unmatched strings.
- AWS, GCP context, GitHub, Slack, Stripe, JWT, PEM, generic assignment, and database URL rules.
- Approximate iyzico, PayTR, Param, Netgsm, and Turkish bank/API assignment rules.
- Fully redacted findings with path, line, rule, and confidence.
- Colored terminal table, JSON, and SARIF 2.1.0 output.
- Glob and regex allowlists, optional YAML configuration, and explicit flag overrides.
- Binary, size, dependency, and common fixture exclusions.

## Install

Go 1.25 or newer is required. From a source checkout:

```sh
go mod download
go build -o bin/sizinti ./cmd/sizinti
go install ./cmd/sizinti
```

On Windows, use `go build -o bin/sizinti.exe ./cmd/sizinti`. `go install` writes to `GOBIN`, or `GOPATH/bin` when `GOBIN` is unset; add that directory to your PATH.

The module path is `github.com/toprakgureli/sizinti`.

## Quick start

```sh
sizinti .
sizinti ../backend --workers 4 --entropy-threshold 4.8
sizinti . --json > ../sizinti-result.json
sizinti . --sarif > ../sizinti-result.sarif
sizinti . --include-fixtures --max-file-size 20971520
sizinti . --config sizinti.example.yaml
```

Write reports outside the scanned directory so the scanner does not read its own output.

| Exit code | Meaning |
| --- | --- |
| 0 | No findings in the scanned scope |
| 1 | Possible credentials found |
| 2 | Invalid input, cancellation, read failure, or output failure |

Errors go to stderr; JSON/SARIF go to stdout. A scan error produces no report, so check the exit code before interpreting an empty file. Use the built executable when checking exact exit codes: `go run` wraps nonzero program exits.

## Example output

The shipped fixture contains the synthetic assignment `iyzico_api_key="synthetic-only"`.

```sh
sizinti testdata/synthetic --no-color
```

```text
FILE       LINE  RULE    CONFIDENCE  SNIPPET
"app.env"  1     iyzico  medium      [REDACTED]
1 finding(s); 1 scanned; 0 skipped entries
```

This command exits with code 1. Reports contain `[REDACTED]` instead of credential values or source lines. File paths remain visible.

## How it works

A producer walks the directory and sends paths through an unbuffered channel. Workers check each file for binary content using a bounded buffer, rewind it, and scan it line by line. One collector owns the findings and sorts them by path, line, and rule.

Specific regex rules run before Turkish provider assignments and generic assignments. Overlapping matches are reported once. Unmatched token-like strings of at least 20 bytes are checked against the entropy threshold and receive low confidence.

Each line must fit in a 1 MiB scanner buffer, including delimiter overhead; longer lines cause an error. File-reading buffers are bounded, while findings and directory entry lists consume additional memory. [DESIGN.md](DESIGN.md) explains the ownership and concurrency model.

## Configuration

Precedence is **defaults → YAML supplied with `--config` → explicitly supplied flags**. YAML files are limited to 1 MiB; unknown or duplicate fields and multiple documents are errors.

| Setting / flag | Default | Meaning |
| --- | --- | --- |
| `workers` / `--workers` | `runtime.NumCPU()` | Worker count, 1–1024 |
| `entropy-threshold` / `--entropy-threshold` | `4.5` | Finite bits/byte threshold, 0–8 |
| `max-file-size` / `--max-file-size` | `10485760` | Maximum file size in bytes |
| `include-fixtures` / `--include-fixtures` | `false` | Include common test/example paths |
| `ignore-file` / `--ignore-file` | `.sizintiignore` | Relative to scan root, or absolute |
| `format` | `table` | YAML: `table`, `json`, or `sarif` |
| `no-color` / `--no-color` | `false` | Disable table color |

`--json` and `--sarif` override the format and are mutually exclusive. Explicit Boolean flags such as `--include-fixtures=false` override YAML. `--ignore-file=` disables ignore loading. A missing default `.sizintiignore` is allowed; a missing differently named file is an error. Color requires character-device output and is disabled by `NO_COLOR` or `--no-color`.

The config path is relative to the current working directory. Ignore-file paths and globs are resolved from the scan root.

## False positives and exclusions

Example `.sizintiignore`:

```text
glob:go.sum
glob:*.sample
glob:config/*.local
regex:synthetic-[a-z]+
```

Blank lines and lines beginning with `#` are ignored. Globs use Go `path.Match` syntax: `*`, `?`, and character classes, with forward slashes on every OS. Patterns without a slash match basenames at any depth; patterns with a slash match root-relative paths. Matching a directory excludes its subtree. Recursive `**`, negation, and Gitignore semantics are not supported.

Regex exceptions match the whole candidate, so allowing one value leaves other credentials on the same line eligible. Keep exceptions narrow and synthetic.

Default exclusions:

- `.git`, `vendor`, and `node_modules` directories.
- `testdata`, `fixtures`, and `examples` directories; `*_test.go`, `*.example`, and `*.sample` files. Use `--include-fixtures` to include them.
- Symlinks, non-regular files, and files above the size limit.
- Files containing control bytes below 9 or between 14 and 31. This binary heuristic also excludes NUL-containing UTF-16 files.

An explicitly selected root is scanned even if named `fixtures` or `testdata`. Each pruned directory counts as one skipped entry. `.gitignore` is not loaded, so untracked `.env` files remain eligible. For audits, review exclusions and size limits alongside `--include-fixtures`.

Higher entropy thresholds reduce noise but miss more credentials; lower thresholds increase sensitivity and false positives. Hashes and random IDs can have high entropy without granting access. Generic assignments can also match harmless expressions. Short values below four bytes and multiline values without a recognized marker may be missed.

## Rules and confidence

| Family | Recognition | Confidence |
| --- | --- | --- |
| AWS | `AKIA`/`ASIA` IDs; 40-character values beside `aws_secret_access_key` | High |
| GCP | JSON `type: service_account`; private keys through PEM | Low for context; high for PEM |
| GitHub | `ghp_` / `gho_` plus 36 alphanumeric characters | High |
| Slack / Stripe | Slack token prefixes; `sk_live_` / `sk_test_` shapes | High |
| JWT | Three token-like segments starting with an `eyJ` header | Medium |
| PEM | Private-key BEGIN markers, including RSA, EC, DSA, OpenSSH, encrypted | High |
| Database URLs | Password-bearing PostgreSQL, MySQL, MongoDB URLs | High |
| Generic | `api_key`, `secret`, `password`, `passwd`, `token`, `client_secret` assignments | Medium |
| Turkish providers | Provider-prefixed assignments | Medium |
| Entropy | Unmatched token-like strings of at least 20 bytes | Low |

Confidence describes the pattern, not whether a credential works. The GCP marker can appear without a private key; PEM detection recognizes the opening marker rather than validating the block; JWTs are shape-matched.

Turkish rules are approximate and open to contributions. They match case-insensitive field names, with underscore/hyphen variations defined in the source:

- iyzico: `iyzico_api_key`, `iyzico_secret_key`.
- PayTR: `paytr_merchant_key`, `paytr_merchant_salt`, `paytr_api_key`.
- Param: `param_client_code`, `param_client_username`, `param_client_password`, `param_guid`, `param_api_key`.
- Netgsm: `netgsm_password`, `netgsm_usercode`, `netgsm_api_key`.
- Bank/API: prefixes `banka`, `bank`, `turkish_bank`, `tr_bank` with `api_key`, `client_secret`, or `password`.

Some identifiers may be public. Different naming conventions can also lose the provider-specific label. Contributions should include provider documentation and synthetic positive/negative examples, never working credentials.

## CI and hooks

Run `sizinti .` as a required pipeline check or from an existing pre-commit hook, preserving its exit code. For GitHub SARIF uploads, scan from the repository root and write the report outside it. Upload with `github/codeql-action/upload-sarif` and `security-events: write`; allow upload after exit code 1 while keeping the scan failed, and skip upload after exit code 2. Uploads are handled by CI, not by sizinti. See [GitHub SARIF support](https://docs.github.com/en/code-security/reference/code-scanning/sarif-files/sarif-support).

The included project workflow runs build/test/vet on Windows, Linux, and macOS, plus linting, formatting checks, and a Linux race-detector run.

## Development

```sh
go build ./...
go test ./...
go test -race ./...
go vet ./...
staticcheck ./...
golangci-lint run
goimports -local github.com/toprakgureli/sizinti -w .
go test ./internal/scanner -run '^$' -bench . -benchmem
```

The race detector needs a supported C toolchain. Tests use synthetic local inputs. GNU Make is optional: `make build`, `make test`, `make lint`, `make run ARGS="."`, `make bench`, and `make fmt` wrap the same commands.

Tool versions: goimports v0.36.0 (`golang.org/x/tools/cmd/goimports`), staticcheck v0.6.1 (`honnef.co/go/tools/cmd/staticcheck`), and golangci-lint v2.12.2.

Recorded package coverage: entropy 100%, allowlist 97.5%, config 95.5%, report 91.9%, rules 89.3%, scanner 88.9%, CLI 88.5%.

One local Windows/amd64 measurement on a Ryzen 7 5800X3D, using one scanner worker:

```text
BenchmarkScan-16    222    5274423 ns/op    19.42 MB/s    239670 B/op    8240 allocs/op
```

This is one local measurement, not a comparison. The `-16` suffix is the benchmark process setting, not the scanner worker count.

Code follows the [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md); [DESIGN.md](DESIGN.md) explains the key design decisions.

## Inspired by

[Gitleaks](https://github.com/gitleaks/gitleaks) and [TruffleHog](https://github.com/trufflesecurity/trufflehog) inspired the workflow. They offer broader coverage; sizinti is my focused implementation, built around readable Go and explicit Turkish provider context rules.

## Roadmap

- Git history scanning.
- Staged-index scanning and an installable pre-commit hook.
- More documented Turkish rules and negative examples.
- Broader credential formats and more precise contextual matching.

## License

[MIT](LICENSE).
