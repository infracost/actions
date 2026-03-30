# Scanner

Scanner is a Go CLI tool that powers the `infracost/actions/scan` composite GitHub Action. It embeds the Infracost CLI as a Go library to scan two directories of infrastructure code (typically from a base branch and a PR branch), calculates a cost diff between them, and posts a comment on the pull request via the Infracost VCS API.

The tool has no external runtime dependencies — the Infracost scanning and diffing logic is imported directly via `go get` rather than shelling out to the CLI. Core scanning logic is shared with the CLI via `github.com/infracost/cli/pkg/scanner`.

## Remaining work

### Make VCS repository public

The `infracost/vcs` repository must be made public before these changes can be merged. The scanner imports it as a Go module dependency, and CI will fail to fetch it while it remains private.

### Composite GitHub Action

Create a `scan/` composite action in this repository that downloads the pre-built scanner binary for the current platform and runs it. This is the user-facing entry point — users will reference `infracost/actions/scan@v1` in their workflows.

### Release workflow

Add a GitHub Actions workflow that builds multi-platform scanner binaries (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64) and attaches them to a GitHub release in this repository. The composite action will download the appropriate binary from the release.

### Integration tests

Add integration tests that exercise the full `Scan()` flow — from dashboard parameter fetching through to VCS comment posting. Tests should download real provider plugins and scan real Terraform fixtures, but mock all external HTTP APIs.

#### Mocking approach

Follow the same patterns used in `infracost/cli`:

- **mockery** for generating mocks from Go interfaces. The scanner has two interfaces to mock: `dashboard.Client` and `events.Client`. Add a `.mockery.yml` at the scanner root and generate mocks into `<package>/mocks/` directories using the testify template.
- **`vcs.VCS` interface** — mock directly using testify since it comes from an external module. This lets tests capture the `comment.Data` passed to `GenerateComment` and the body passed to `PostComment` without hitting a real GitHub API.
- **Centralized test config** — create an `internal/config/testing/` package (mirroring the CLI's `internal/config/testing/config.go`) that returns a `Config` struct pre-wired with mock clients. This avoids duplicating mock setup across test files.
- **Terraform fixtures** in `internal/config/testdata/` with simple resources (e.g. `aws_instance`) where expected costs are known. Separate `base/` and `head/` directories to produce predictable diffs.
- **`t.TempDir()`** for all file I/O, **`t.Setenv()`** for environment variables, and testify's automatic `AssertExpectations` via `t.Cleanup()`.

#### Test cases

| Test case | What it validates |
|---|---|
| Basic cost diff | Base has one instance, head adds another — comment shows cost increase |
| No changes | Identical base and head — comment shows no diff |
| Guardrail triggered | Dashboard returns a guardrail that head violates — comment includes warning |
| Guardrail suppressed | Guardrail already triggered in base — suppressed in comment |
| Policy evaluation | Dashboard returns FinOps and tag policies — comment includes results |
| Usage defaults | Dashboard returns usage defaults — applied to cost estimates |
| Dashboard error | Dashboard API returns error — `Scan()` returns error |
| Single project filter | `--project` flag filters to one project — only that project scanned |
