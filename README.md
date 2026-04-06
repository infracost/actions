# Infracost GitHub Actions

This repository contains GitHub Actions for running Infracost in your CI pipeline to see cloud cost estimates for Terraform in pull requests.

We recommend using the [Infracost GitHub App](https://www.infracost.io/docs/integrations/github_app/) where possible as it's simpler to set up and faster to run. If your organization doesn't allow GitHub App installations, use the actions below as an alternative.

## Actions

### [`diff`](diff/)

Computes cost differences between two branches, posts a PR comment, and manages PR status in the Infracost dashboard. Handles the full PR lifecycle — scans on open/sync/reopen, updates status on close.

### [`scan`](scan/)

Scans a single directory and uploads baseline cost data to the Infracost dashboard. Use this on pushes to your default branch to keep the dashboard up to date.

### Full example

```yaml
on:
  pull_request:
    types: [opened, synchronize, closed, reopened]
  push:
    branches: [main]

permissions:
  contents: read
  pull-requests: write

jobs:
  diff:
    if: github.event_name == 'pull_request'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        if: github.event.action != 'closed'
        with:
          path: head

      - uses: actions/checkout@v4
        if: github.event.action != 'closed'
        with:
          ref: ${{ github.event.pull_request.base.ref }}
          path: base

      - uses: infracost/actions/diff@v4
        with:
          api-key: ${{ secrets.INFRACOST_API_KEY }}
          base-path: base
          head-path: head

  scan:
    if: github.event_name == 'push'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: infracost/actions/scan@v4
        with:
          api-key: ${{ secrets.INFRACOST_API_KEY }}
          path: .
```

See the [`diff` README](diff/README.md) and [`scan` README](scan/README.md) for the full list of inputs.

### [`setup`](setup/) (legacy)

Installs the Infracost CLI into your workflow. This requires you to manually run `infracost breakdown`, `infracost diff`, and `infracost comment` as separate steps. Use [`diff`](diff/) and [`scan`](scan/) instead for a simpler setup.

<details>
<summary>Legacy setup example</summary>

```yaml
- name: Setup Infracost
  uses: infracost/actions/setup@v3
  with:
    api-key: ${{ secrets.INFRACOST_API_KEY }}

- name: Checkout base branch
  uses: actions/checkout@v4
  with:
    ref: '${{ github.event.pull_request.base.ref }}'

- name: Generate Infracost cost estimate baseline
  run: |
    infracost breakdown --path=. \
                        --format=json \
                        --out-file=/tmp/infracost-base.json

- name: Checkout PR branch
  uses: actions/checkout@v4

- name: Generate Infracost diff
  run: |
    infracost diff --path=. \
                    --format=json \
                    --compare-to=/tmp/infracost-base.json \
                    --out-file=/tmp/infracost.json

- name: Post Infracost comment
  run: |
    infracost comment github --path=/tmp/infracost.json \
                             --repo=$GITHUB_REPOSITORY \
                             --github-token=${{ github.token }} \
                             --pull-request=${{ github.event.pull_request.number }} \
                             --behavior=update
```

</details>

## Contributing

We welcome contributions! Please start by opening a thread in [GitHub Discussions](https://github.com/infracost/infracost/discussions) to discuss your idea before submitting a PR.

## Bugs and feedback

If you run into any issues or have feedback, please open a thread in [GitHub Discussions](https://github.com/infracost/infracost/discussions). We'd love to hear from you!

## License

[Apache License 2.0](https://choosealicense.com/licenses/apache-2.0/)
