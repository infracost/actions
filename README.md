# Infracost GitHub Actions

This repository contains GitHub Actions for running Infracost in your CI pipeline to see cloud cost estimates for Terraform in pull requests.

We recommend using the [Infracost GitHub App](https://www.infracost.io/docs/integrations/github_app/) where possible as it's simpler to set up and faster to run. If your organization doesn't allow GitHub App installations, use the `scan` action below as an alternative.

## Actions

### [`scan`](scan/)

The recommended action when using GitHub Actions. Scans two checkouts of your infrastructure code (base branch and PR branch), calculates a cost diff, posts a PR comment, and manages PR status in the Infracost dashboard — all in a single step.

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
      - uses: actions/checkout@v4
        if: github.event.action != 'closed'
        with:
          path: head

      - uses: actions/checkout@v4
        if: github.event.action != 'closed'
        with:
          ref: ${{ github.event.pull_request.base.ref }}
          path: base

      - uses: infracost/actions/scan@v4
        with:
          api-key: ${{ secrets.INFRACOST_API_KEY }}
          base-path: base
          head-path: head
```

See the [`scan` README](scan/README.md) for the full list of inputs.

### [`setup`](setup/) (legacy)

Installs the Infracost CLI into your workflow. This requires you to manually run `infracost breakdown`, `infracost diff`, and `infracost comment` as separate steps. Use [`scan`](scan/) instead for a simpler setup.

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

Issues and pull requests are welcome! For major changes, including interface changes, please open an issue first to discuss what you would like to change.

## License

[Apache License 2.0](https://choosealicense.com/licenses/apache-2.0/)