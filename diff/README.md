# Infracost Diff

A GitHub Action that computes cost differences between two branches of infrastructure code, posts a PR comment with the diff, and manages PR status in the Infracost dashboard.

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
      # Checkout steps are skipped on closed PRs since diffing is not needed.
      - uses: actions/checkout@v4
        if: github.event.action != 'closed'
        with:
          path: head

      - uses: actions/checkout@v4
        if: github.event.action != 'closed'
        with:
          ref: ${{ github.event.pull_request.base.ref }}
          path: base

      # The diff action runs on every event. On closed PRs it skips diffing
      # and only updates the PR status in the Infracost dashboard.
      - uses: infracost/actions/diff@v4
        with:
          api-key: ${{ secrets.INFRACOST_API_KEY }}
          base-path: base
          head-path: head
```

The action detects the event type automatically:
- **opened / synchronize / reopened** — scans both checkouts, posts a PR comment, uploads results to the Infracost dashboard, and sets the PR status to `OPEN`
- **closed** — skips scanning and updates the PR status in the Infracost dashboard to `MERGED` or `CLOSED`

### Disabling the dashboard

To skip all dashboard interactions (addRun and PR status updates), set the `INFRACOST_CI_DISABLE_DASHBOARD` environment variable:

```yaml
- uses: infracost/actions/diff@v4
  env:
    INFRACOST_CI_DISABLE_DASHBOARD: "true"
  with:
    api-key: ${{ secrets.INFRACOST_API_KEY }}
    base-path: base
    head-path: head
```

### Manual PR status

To set the PR status explicitly (e.g. from a `workflow_dispatch`):

```yaml
- uses: infracost/actions/diff@v4
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
| `base-path` | No | | Path to the base branch checkout (required for diffing) |
| `head-path` | No | | Path to the head (PR) branch checkout (required for diffing) |
| `version` | No | `latest` | Scanner version to use (e.g. `0.1.0`) |
| `project` | No | | Filter scanning to a single project |
| `pr-status` | No | Auto-detected | Explicitly set PR status (`OPEN`, `MERGED`, `CLOSED`) |
| `github-token` | No | `github.token` | GitHub API token for posting PR comments |
| `github-owner` | No | Current owner | GitHub repository owner |
| `github-repo` | No | Current repo | GitHub repository name |
| `pr-number` | No | Current PR | Pull request number to comment on |
| `repo-url` | No | Current repo URL | Repository URL for source links in comments |

## Requirements

- A PR number is required. This is derived automatically on `pull_request` events, or can be set explicitly via the `pr-number` input.
- Both `git` and `gh` (GitHub CLI) must be available on the runner. GitHub-hosted runners include them by default; self-hosted runners may need to install them.