# Scanner

> **Frozen.** This source moved to [infracost/ci](https://github.com/infracost/ci) at `46b6838` (FIX-716) and ships as `ghcr.io/infracost/ci`, which `diff@v5` and `scan@v5` run. Make changes there. A hotfix for the `scanner/v*` releases that `@v4` still resolves is a cherry-pick into this directory **plus restoring `.github/workflows/scanner_release.yml`**, which was deleted with `v5` and is the only thing that builds and uploads those assets. This copy is deleted once `infracost/ci` has cut a release after `v5.0.0` and 30 days have passed with no cherry-pick (FIX-726).

Scanner is the Go CLI that powers the Infracost GitHub Actions. It embeds the Infracost CLI as a library to scan directories of infrastructure code, calculate cost diffs, and post comments on pull requests.

The tool requires `git` at runtime to derive commit SHAs, branch names, and commit metadata from the checkout directories. Scanning and diffing logic is imported directly via `github.com/infracost/cli/pkg/scanner` rather than shelling out to the Infracost CLI.

## Commands

- `scanner diff` — Scan base and head branches, compute a cost diff, post a PR comment, and upload results to the Infracost dashboard. Powers [`infracost/actions/diff`](../../diff).
- `scanner scan` — Scan a single directory and upload baseline results to the Infracost dashboard. Powers [`infracost/actions/scan`](../../scan).
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

This repo no longer publishes scanner releases; `.github/workflows/scanner_release.yml` was removed with `diff@v5`, so a `scanner/v*` tag now builds nothing until that workflow is restored. Releases ship from [infracost/ci](https://github.com/infracost/ci). The existing `scanner/v*` releases stay published for `@v4`.