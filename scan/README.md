# Infracost Scan

A GitHub Action that scans infrastructure code and uploads baseline cost data to the Infracost dashboard. Use this on pushes to your default branch to keep the dashboard's view of your repository up to date.

## Usage

```yaml
on:
  push:
    branches: [main]

jobs:
  infracost:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: infracost/actions/scan@v4
        with:
          api-key: ${{ secrets.INFRACOST_API_KEY }}
          path: .
```

## Inputs

| Input | Required | Default | Description |
|---|---|---|---|
| `api-key` | Yes | | Infracost API key for authentication |
| `path` | Yes | | Path to the directory to scan |
| `version` | No | `latest` | Scanner version to use (e.g. `0.1.0`) |
| `project` | No | | Filter scanning to a single project |
| `repo-url` | No | Current repo URL | Repository URL for metadata |
| `github-token` | No | `github.token` | GitHub API token used to download the scanner binary |

## Requirements

- Both `git` and `gh` (GitHub CLI) must be available on the runner. GitHub-hosted runners include them by default; self-hosted runners may need to install them.