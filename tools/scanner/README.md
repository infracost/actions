# Scanner

Scanner is a Go CLI tool that powers the `infracost/actions/scan` composite GitHub Action. It embeds the Infracost CLI as a Go library to scan two directories of infrastructure code (typically from a base branch and a PR branch), calculates a cost diff between them, and posts a comment on the pull request via the Infracost VCS API.

The tool has no external runtime dependencies — the Infracost scanning and diffing logic is imported directly via `go get` rather than shelling out to the CLI. Core scanning logic is shared with the CLI via `github.com/infracost/cli/pkg/scanner`.

## Remaining work

### Composite GitHub Action

Create a `scan/` composite action in this repository that downloads the pre-built scanner binary for the current platform and runs it. This is the user-facing entry point — users will reference `infracost/actions/scan@v1` in their workflows.

### Release workflow

Add a GitHub Actions workflow that builds multi-platform scanner binaries (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64) and attaches them to a GitHub release in this repository. The composite action will download the appropriate binary from the release.
