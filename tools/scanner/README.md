# Scanner

Scanner is the Go CLI that powers the [`infracost/actions/scan`](../../scan) composite GitHub Action. It embeds the Infracost CLI as a library to scan two directories of infrastructure code (typically a base branch and a PR branch), calculates a cost diff, and posts a comment on the pull request.

The tool has no external runtime dependencies — scanning and diffing logic is imported directly via `github.com/infracost/cli/pkg/scanner` rather than shelling out to the CLI.

## Development

```bash
make build          # Build the binary
make test           # Run all tests
make test-unit      # Run unit tests only (skips integration tests)
make test-integration # Run integration tests (requires INFRACOST_CLI_AUTHENTICATION_TOKEN)
make lint           # Run golangci-lint
make mocks          # Regenerate mockery mocks
```

## Releasing

Push a tag matching `scanner/v*.*.*` to trigger the [release workflow](../../.github/workflows/scanner_release.yml), which builds multi-platform binaries and creates a GitHub release. The version is set via `-ldflags` at build time.
