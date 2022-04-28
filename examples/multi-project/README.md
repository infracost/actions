# Multi-project

These examples show how to run Infracost actions against a multi-project setup using either an [Infracost config file](https://www.infracost.io/docs/multi_project/config_file) or a GitHub Actions build matrix.

## Using an Infracost config file

This example shows how to run Infracost actions with multiple Terraform projects using an [Infracost config file](https://www.infracost.io/docs/multi_project/config_file).

[//]: <> (BEGIN EXAMPLE)
```yml
name: Multi-project config file
on: [pull_request]

jobs:
  multi-project-config-file:
    name: Multi-project config file
    runs-on: ubuntu-latest

    steps:
      # Checkout the branch you want Infracost to compare costs against.This example is using the 
      # target PR branch.
      - name: Checkout base branch
        uses: actions/checkout@v2
        with:
          ref: '${{ github.event.pull_request.base.ref }}'

      - name: Setup Infracost
        uses: infracost/actions/setup@v2
        with:
          api-key: ${{ secrets.INFRACOST_API_KEY }}

      # Generate an Infracost output JSON from the comparison branch, so that Infracost can compare the cost difference.
      - name: Generate Infracost cost snapshot
        run: |
          infracost breakdown --config-file=examples/multi-project/code/infracost.yml \
                              --format=json \
                              --out-file=/tmp/prior.json

      - name: Checkout pr branch
        uses: actions/checkout@v2

      - name: Run Infracost
        run: |
          infracost diff --config-file=examples/multi-project/code/infracost.yml \
                              --format=json \
                              --compare-to=/tmp/prior.json \
                              --out-file=/tmp/infracost.json

      - name: Post Infracost comment
        run: |
          # Posts a comment to the PR using the 'update' behavior.
          # This creates a single comment and updates it. The "quietest" option.
          # The other valid behaviors are:
          #   delete-and-new - Delete previous comments and create a new one.
          #   hide-and-new - Minimize previous comments and create a new one.
          #   new - Create a new cost estimate comment on every push.
          # See https://www.infracost.io/docs/features/cli_commands/#comment-on-pull-requests for other options.
          infracost comment github --path /tmp/infracost.json \
                                   --repo $GITHUB_REPOSITORY \
                                   --github-token ${{github.token}} \
                                   --pull-request ${{github.event.pull_request.number}} \
                                   --behavior update
```
[//]: <> (END EXAMPLE)

## Using GitHub Actions build matrix

This example shows how to run Infracost actions with multiple Terraform projects using a GitHub Actions build matrix. The first job uses a build matrix to generate multiple Infracost output JSON files and upload them as artifacts. The second job downloads these JSON files and posts a combined comment to the PR using `infracost comment` glob support.

[//]: <> (BEGIN EXAMPLE)
```yml
name: Multi-project matrix
on: [pull_request]

jobs:
  multi-project-matrix:
    name: Multi-project matrix
    runs-on: ubuntu-latest

    strategy:
      matrix:
        include:
          - dir: dev
          - dir: prod

    steps:
      # Checkout the branch you want Infracost to compare costs against. This example is using the 
      # target PR branch.
      - name: Checkout base branch
        uses: actions/checkout@v2
        with:
          ref: '${{ github.event.pull_request.base.ref }}'

      - name: Setup Infracost
        uses: infracost/actions/setup@v2
        with:
          api-key: ${{ secrets.INFRACOST_API_KEY }}

      # Generate an Infracost output JSON from the comparison branch, so that Infracost can compare the cost difference.
      - name: Generate Infracost cost snapshot
        run: |
          infracost breakdown --path=examples/multi-project/code/${{ matrix.dir }} \
                              --format=json \
                              --out-file=/tmp/prior.json
          
      - name: Checkout pr branch
        uses: actions/checkout@v2

      - name: Run Infracost
        run: |
          infracost diff --path=examples/multi-project/code/${{ matrix.dir }} \
                              --format json \
                              --compare-to=/tmp/prior.json \
                              --out-file=/tmp/infracost_${{ matrix.dir }}.json

      - name: Upload Infracost breakdown
        uses: actions/upload-artifact@v2
        with:
          name: infracost_jsons
          path: /tmp/infracost_${{ matrix.dir }}.json

  multi-project-matrix-merge:
    name: Multi-project matrix merge
    runs-on: ubuntu-latest
    needs: [multi-project-matrix]

    steps:
      - uses: actions/checkout@v2

      - name: Download all Infracost breakdown files
        uses: actions/download-artifact@v2
        with:
          path: /tmp

      - name: Setup Infracost
        uses: infracost/actions/setup@v2
        with:
          api-key: ${{ secrets.INFRACOST_API_KEY }}

      - name: Post Infracost comment
        run: |
          # Posts a comment to the PR using the 'update' behavior.
          # This creates a single comment and updates it. The "quietest" option.
          # The other valid behaviors are:
          #   delete-and-new - Delete previous comments and create a new one.
          #   hide-and-new - Minimize previous comments and create a new one.
          #   new - Create a new cost estimate comment on every push.
          # See https://www.infracost.io/docs/features/cli_commands/#comment-on-pull-requests for other options.
          infracost comment github --path "/tmp/infracost_jsons/*.json" \
                                   --repo $GITHUB_REPOSITORY \
                                   --github-token ${{github.token}} \
                                   --pull-request ${{github.event.pull_request.number}} \
                                   --behavior update
```
[//]: <> (END EXAMPLE)
