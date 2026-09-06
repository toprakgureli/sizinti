# Design

Sizinti scans a local working tree for possible credentials. The design keeps file reading bounded, goroutine lifetimes explicit, and secret values out of reports. All detection happens locally.

## Package and type ownership

```text
cmd/sizinti -> cli
cli -> config, scanner, report, rules, allowlist
config -> scanner
report -> scanner
scanner -> rules, entropy, allowlist
```

`rules.Match` belongs to detection. It holds byte offsets into one line, a rule identifier, and confidence. The scanner uses those offsets to apply allowlist exceptions while the source line is available.

`scanner.Finding` belongs to scanning. It holds a root-relative path, one-based line number, rule, confidence, and the literal `[REDACTED]` snippet. Source lines and credential values stay out of this boundary, which also prevents neighboring secrets from leaking through a context snippet.

`scanner.Result` owns the collected findings and counts. Reports consume it; rules know nothing about scanner or report. The dependency direction stays `report -> scanner -> rules`, with no shared catch-all model package.

`config.Config` defines the CLI-facing YAML schema and converts to scalar `scanner.Settings`. Optional scanner settings use functional options. Standard `io.Reader`, `io.Writer`, and `error` interfaces cover the I/O boundaries.

Compiled regexes and allowlists have private fields and no mutation methods, so workers can share them. Cobra receives a cloned argument slice; matches are fresh slices; settings are copied by value. Reports leave caller-owned findings unchanged. These ownership rules avoid mutable globals and shared-state locks.

## Concurrency model

`Scanner.Scan` validates the root, derives a cancelable context, and creates two unbuffered channels plus a capacity-one error channel.

1. The producer walks with `filepath.WalkDir`, applies exclusions, and sends eligible paths. It closes the path channel on every exit.
2. Exactly `Workers` workers consume paths. Each opens at most one file at a time and sends redacted findings and counters.
3. A `sync.WaitGroup` tracks the producer and workers. A closing goroutine waits for them, then closes the result channel.
4. The caller collects results until channel closure and sorts them before returning.

Unbuffered channels provide backpressure. Every potentially blocking send also checks cancellation. Workers waiting for paths are released when the producer closes that channel. Results close only after every sender has finished.

The first failure enters the capacity-one error channel before cancellation. Later error sends are nonblocking. The collector keeps draining while work shuts down, then `Scan` returns the first failure or the parent's cancellation error. The library can return partial findings with an error; the CLI emits no report in that case and returns 2.

Only the collector changes findings and counters, so no mutex is needed. Sorting by path, line, and rule makes output independent of worker completion order. Identical redacted findings can remain repeated.

Cancellation is cooperative: traversal, classification, line scanning, and channel sends check the context. An operating-system read already blocked on a filesystem can still delay shutdown. The intended input is a stable local checkout. Symlinks and special files are skipped, and file identity is checked after opening; concurrent hostile filesystem changes are outside that guarantee.

## File processing and memory

Files over the byte limit are skipped. Each eligible file is first read in bounded chunks to check for binary control bytes, then rewound and scanned line by line. This extra sequential read avoids reporting findings before discovering binary data near the end. NUL-containing UTF-16 files are excluded by the binary heuristic.

A worker uses a 32 KiB classification buffer and a line buffer that starts at 64 KiB and can grow to 1 MiB. Matching allocations depend on candidates within that bounded line. Lines that cannot fit cause errors. Limited readers also catch growth beyond the size limit. Files should remain stable across the two passes.

Total memory is approximately `O(workers * bounded-line-work + findings + largest-directory-entry-list)`. The collector retains findings for sorting, `filepath.WalkDir` reads directory entries, and report encoding adds result-sized allocations. The constant bound applies to file-reading buffers.

Deferred cleanup closes files on success, errors, and cancellation, preserving close errors. The CLI leaves caller-owned output streams open. Signal cleanup finishes in `run` before the single `os.Exit` call in `main`.

## Detection and priority

Rules compile once during construction with `regexp.Compile`; compilation failures return errors.

Priority is recognizable global formats, Turkish provider assignments, then generic assignments. The first rule owning a span wins over overlapping matches. Separate credentials on one line remain separate. For example, a recognizable GitHub token inside an assignment keeps the GitHub label.

A definition selects its whole match or a capture group as the candidate span. Database URL rules isolate the password. The scanner applies whole-candidate allowlist matching, then checks remaining token-like strings for entropy. Known spans stay excluded from entropy even when allowlisted, so an exception cannot reappear under a different rule.

PEM detection recognizes a private-key BEGIN marker, including one inside an escaped JSON string. GCP's service-account type marker is a low-confidence contextual hint. JWTs are shape-matched. Confidence describes recognition strength; no provider request tests whether a value works.

## Entropy and thresholds

For each byte value, `p` is its frequency divided by candidate length. Shannon entropy is `H = -sum(p * log2(p))`, in bits per byte. Empty and repeated-byte strings score zero; four equally frequent byte values score two. A 256-entry frequency array keeps counting storage fixed.

Candidates contain ASCII letters, digits, underscore, plus, slash, equals, or hyphen and must be at least 20 bytes long. The calculation is byte-based. Thresholds must be finite and between zero and eight. Since the candidate alphabet is smaller than 256 symbols, a threshold of eight effectively disables entropy findings while retaining regex detection.

The default is 4.5. Raising it reduces noise and misses more secrets; lowering it catches more strings and increases false positives. Hashes can be random without granting access, while real passwords can be predictable. Entropy therefore complements format and assignment rules with low-confidence findings.

## False-positive controls

- Path policy excludes dependencies and common fixtures. `--include-fixtures` enables test/example paths for audits.
- `.sizintiignore` globs exclude chosen paths using Go `path.Match` syntax and forward slashes. Recursive doublestar and Gitignore negation are unsupported.
- Whole-candidate regex exceptions allow narrow synthetic values while leaving neighboring credentials eligible for detection.

Exclusions also create blind spots, so users should review them with the skipped-entry count. Predictable values are not automatically considered safe. The safe fixture uses ordinary settings, an environment reference, and a passwordless database URL; its test checks both that it was read and that it produced zero findings.

Turkish rules use provider-prefixed field names as approximate clues. Some fields, such as usercodes, may be public; alternative naming can also miss a provider label. Improvements should pair provider documentation with synthetic positive and negative examples.

## Reports and integration

Reports consume redacted findings. Table paths escape control characters; machine output is one JSON document on stdout, with errors on stderr. SARIF uses stable rule IDs, percent-encoded root-relative URIs, and one-based lines. It includes neither source snippets nor credential-derived fingerprints.

For GitHub uploads, scan from the repository root so URIs map to source files. CI handles upload permissions and can compute fingerprints from the checkout. Large reports may need partitioning to fit GitHub's ingestion limits.

An existing pre-commit hook can run the working-tree scan today. Staged-index support and hook installation are planned separately. Exit code 0 describes the scanned scope; excluded files and Git history need their own review.

## Testing

Table-driven tests cover entropy values, every rule family, negative examples, allowlists, configuration, scanner behavior, and CLI exit codes. Output tests check redaction. Concurrency tests compare worker counts and cancel a worker during result delivery. Filesystem tests cover binary, oversized, fixture, and long-line behavior.

CI builds and tests on Windows, Linux, and macOS, with the race detector on Linux. Formatting, `go vet`, staticcheck through golangci-lint, and explicit lint rules support review. Ownership, cancellation, and error propagation remain useful manual review points.

The benchmark creates input before timing, scans a fixed file with one synthetic finding, reports allocations, and verifies detection. The README records coverage and one local measurement.
