# Multi-Terraform workspace

This example shows how to run Infracost actions against a Terraform project that uses multiple workspaces using an [Infracost config file](https://www.infracost.io/docs/multi_project/config_file).

[//]: <> (BEGIN EXAMPLE)
```yml
name: Multi-terraform workspace config file
on: [pull_request]

jobs:
  multi-terraform-workspace-config-file:
    name: Multi-Terraform workspace config file
    runs-on: ubuntu-latest

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
          infracost breakdown --config-file=examples/multi-terraform-workspace/code/infracost.yml \
                              --format=json \
                              --out-file=/tmp/prior.json

      - name: Checkout pr branch
        uses: actions/checkout@v2

      - name: Run Infracost
        run: |
          infracost diff --config-file=examples/multi-terraform-workspace/code/infracost.yml \
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
