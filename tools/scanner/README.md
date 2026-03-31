# Scanner

Scanner is the Go CLI that powers the Infracost GitHub Actions. It embeds the Infracost CLI as a library to scan directories of infrastructure code, calculate cost diffs, and post comments on pull requests.

The tool has no external runtime dependencies — scanning and diffing logic is imported directly via `github.com/infracost/cli/pkg/scanner` rather than shelling out to the CLI.

## Commands

- `scanner diff` — Scan base and head branches, compute a cost diff, post a PR comment, and upload results to the Infracost dashboard.
- `scanner status` — Update the pull request status in the Infracost dashboard (OPEN, MERGED, CLOSED).

## Development

```bash
make build            # Build the binary
make test             # Run all tests
make test-unit        # Run unit tests only (skips integration tests)
make test-integration # Run integration tests (requires INFRACOST_CLI_AUTHENTICATION_TOKEN)
make lint             # Run golangci-lint
make mocks            # Regenerate mockery mocks
```

## Releasing

Push a tag matching `scanner/v*.*.*` to trigger the [release workflow](../../.github/workflows/scanner_release.yml), which builds multi-platform binaries and creates a GitHub release. The version is set via `-ldflags` at build time.