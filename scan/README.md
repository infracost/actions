# Infracost Scan

A GitHub Action that scans infrastructure code for cost changes between two branches, posts a PR comment with the diff, and manages PR status in the Infracost dashboard.

## Usage

Your workflow must have `pull-requests: write` permission to post PR comments.

```yaml
on:
  pull_request:
    types: [opened, synchronize, closed, reopened]

permissions:
  contents: read
  pull-requests: write

jobs:
  infracost:
    runs-on: ubuntu-latest
    steps:
      # Checkout steps are skipped on closed PRs since scanning is not needed.
      - uses: actions/checkout@v4
        if: github.event.action != 'closed'
        with:
          path: head

      - uses: actions/checkout@v4
        if: github.event.action != 'closed'
        with:
          ref: ${{ github.event.pull_request.base.ref }}
          path: base

      # The scan action runs on every event. On closed PRs it skips scanning
      # and only updates the PR status in the Infracost dashboard.
      - uses: infracost/actions/scan@v4
        with:
          api-key: ${{ secrets.INFRACOST_API_KEY }}
          base-path: base
          head-path: head
```

The action detects the event type automatically:
- **opened / synchronize / reopened** — scans both checkouts, posts a PR comment, uploads results to the Infracost dashboard, and sets the PR status to `OPEN`
- **closed** — skips scanning and updates the PR status in the Infracost dashboard to `MERGED` or `CLOSED`

### Disabling the dashboard

To skip all dashboard interactions (addRun and PR status updates), set `enable-dashboard` to `false`:

```yaml
- uses: infracost/actions/scan@v4
  with:
    api-key: ${{ secrets.INFRACOST_API_KEY }}
    base-path: base
    head-path: head
    enable-dashboard: "false"
```

### Manual PR status

To set the PR status explicitly (e.g. from a `workflow_dispatch`):

```yaml
- uses: infracost/actions/scan@v4
  with:
    api-key: ${{ secrets.INFRACOST_API_KEY }}
    pr-status: MERGED
    pr-number: "123"
```

When `pr-status` is set and `base-path`/`head-path` are omitted, the action only updates the PR status without scanning.

## Inputs

| Input | Required | Default | Description |
|---|---|---|---|
| `api-key` | Yes | | Infracost API key for authentication |
| `base-path` | No | | Path to the base branch checkout (required for scanning) |
| `head-path` | No | | Path to the head (PR) branch checkout (required for scanning) |
| `version` | No | `latest` | Scanner version to use (e.g. `0.1.0`) |
| `project` | No | | Filter scanning to a single project |
| `branch` | No | PR base ref | Branch name used for policy filtering |
| `enable-dashboard` | No | `true` | Upload results and manage PR status in the dashboard |
| `pr-status` | No | Auto-detected | Explicitly set PR status (`OPEN`, `MERGED`, `CLOSED`) |
| `github-token` | No | `github.token` | GitHub API token for posting PR comments |
| `github-owner` | No | Current owner | GitHub repository owner |
| `github-repo` | No | Current repo | GitHub repository name |
| `pr-number` | No | Current PR | Pull request number to comment on |
| `commit-sha` | No | `GITHUB_SHA` | Head commit SHA for source links |
| `repo-url` | No | Current repo URL | Repository URL for source links in comments |

## Requirements

- A PR number is required. This is derived automatically on `pull_request` events, or can be set explicitly via the `pr-number` input.
- The `gh` CLI must be available on the runner. GitHub-hosted runners include it by default; self-hosted runners may need to [install it](https://github.com/cli/cli#installation).