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

      - uses: infracost/actions/scan@v5
        with:
          api-key: ${{ secrets.INFRACOST_API_KEY }}
          path: .
```

## Inputs

| Input | Required | Default | Description |
|---|---|---|---|
| `api-key` | Yes | | Infracost API key for authentication |
| `path` | Yes | | Path to the directory to scan |
| `version` | No | `2` | Tag of the [`ghcr.io/infracost/ci`](https://github.com/infracost/ci) image to run (e.g. `2`, `2.0.1`, `latest`, or `sha256:<digest>`). Must be `2.0.0` or later; use `scan@v4` for older scanner releases |
| `project` | No | | Filter scanning to a single project |
| `repo-url` | No | Current repo URL | Repository URL for metadata |

`2` is the recommended `version`. The image bakes its parser plugins in, so a parser update only reaches you through a new image; `infracost/ci` cuts a patch release whenever the plugins drift, and `2` picks it up. Pinning `2.0.1` freezes the parsers until you edit the workflow, and a `sha256:` digest freezes them hardest.

## Requirements

- Docker must be available on the runner. GitHub-hosted Linux runners include it; macOS runners do not.
- Only `$GITHUB_WORKSPACE` is mounted into the container, so `path` must be inside it. For checkouts elsewhere, macOS runners, or runners without Docker, use the `container:` job or binary install documented in [infracost/ci](https://github.com/infracost/ci#getting-started).