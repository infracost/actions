# Multi-project using config file

This example shows how to run Infracost in GitHub Actions with multiple Terraform projects using a config file. The [config file]((https://www.infracost.io/docs/config_file/)) can be used to specify variable files or the Terraform workspaces for different projects.

[//]: <> (BEGIN EXAMPLE)
```yml
name: Multi-project config file
on: [pull_request]

jobs:
  multi-project-config-file:
    name: Multi-project config file
    runs-on: ubuntu-latest
    env:
      TF_ROOT: examples/multi-project-config-file/code

    steps:
      - name: Setup Infracost
        uses: infracost/actions/setup@v2
        with:
          api-key: ${{ secrets.INFRACOST_API_KEY }}

      # Checkout the branch you want Infracost to compare costs against.This example is using the
      # target PR branch.
      - name: Checkout base branch
        uses: actions/checkout@v2
        with:
          ref: '${{ github.event.pull_request.base.ref }}'

      # Generate an Infracost output JSON from the comparison branch, so that Infracost can compare the cost difference.
      - name: Generate Infracost cost snapshot
        run: |
          infracost breakdown --config-file=${TF_ROOT}/infracost.yml \
                              --format=json \
                              --out-file=/tmp/infracost-base.json

      - name: Checkout pr branch
        uses: actions/checkout@v2

      - name: Run Infracost
        run: |
          infracost diff --config-file=${TF_ROOT}/infracost.yml \
                              --format=json \
                              --compare-to=/tmp/infracost-base.json \
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
