# Infracost Scan

A GitHub Action that scans infrastructure code for cost changes between two branches and posts a PR comment with the diff.

## Usage

Your workflow must have `pull-requests: write` permission to post PR comments.

```yaml
permissions:
  contents: read
  pull-requests: write

steps:
  - uses: actions/checkout@v4
    with:
      path: head

  - uses: actions/checkout@v4
    with:
      ref: ${{ github.event.pull_request.base.ref }}
      path: base

  - uses: infracost/actions/scan@v4
    with:
      api-key: ${{ secrets.INFRACOST_API_KEY }}
      base-path: base
      head-path: head
```

## Inputs

| Input | Required | Default | Description |
|---|---|---|---|
| `api-key` | Yes | | Infracost API key for authentication |
| `base-path` | Yes | | Path to the base branch checkout |
| `head-path` | Yes | | Path to the head (PR) branch checkout |
| `version` | No | `latest` | Scanner version to use (e.g. `0.1.0`) |
| `project` | No | | Filter scanning to a single project |
| `branch` | No | PR head ref | Branch name used for policy filtering |
| `github-token` | No | `github.token` | GitHub API token for posting PR comments |
| `github-owner` | No | Current owner | GitHub repository owner |
| `github-repo` | No | Current repo | GitHub repository name |
| `pr-number` | No | Current PR | Pull request number to comment on |
| `commit-sha` | No | `GITHUB_SHA` | Head commit SHA for source links |
| `repo-url` | No | Current repo URL | Repository URL for source links in comments |

## Requirements

- A PR number is required to post comments. This is derived automatically on `pull_request` events, or can be set explicitly via the `pr-number` input.
- The `gh` CLI must be available on the runner. GitHub-hosted runners include it by default; self-hosted runners may need to [install it](https://github.com/cli/cli#installation).