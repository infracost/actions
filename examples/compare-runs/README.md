# Compare Runs Example

This example shows how to compare runs from different branches of the same Terraform project using Infracost's `--compare-to` flag.

The flag accepts Infracost JSON file that can be generated using `--format json --out-file file.json` flags.

[//]: <> (BEGIN EXAMPLE)
```yml
name: Compare runs
on: [pull_request]

jobs:
  compare-runs:
    name: Compare runs
    runs-on: ubuntu-latest

    steps:
      - name: Setup Infracost
        uses: infracost/actions/setup@v1
        with:
          api-key: ${{ secrets.INFRACOST_API_KEY }}

      - name: Checkout compare branch
        uses: actions/checkout@v2
        with:
          path: compare_branch
          ref: master # This can be any branch

      - name: Run Infracost on compare branch
        run: infracost breakdown --path=compare_branch/examples/compare-runs/code --terraform-parse-hcl --format=json --out-file=/tmp/infracost_compare.json

      - name: Checkout current branch
        uses: actions/checkout@v2

      - name: Compare Infracost runs between current and compare branches
        run: |
          infracost breakdown --path=examples/compare-runs/code --terraform-parse-hcl --format=json --out-file=/tmp/infracost.json
          diff <(jq --sort-keys . /tmp/infracost.json) <(jq --sort-keys . /tmp/infracost_compare.json)
```
[//]: <> (END EXAMPLE)
