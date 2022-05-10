# Terraform directory

This example shows how to run Infracost actions with a Terraform project containing HCL code.

[//]: <> (BEGIN EXAMPLE)
```yml
name: Terraform directory
on: [pull_request]

jobs:
  terraform-directory:
    name: Terraform directory
    runs-on: ubuntu-latest

    steps:
      - name: Setup Infracost
        uses: infracost/actions/setup@v2
        with:
          api-key: ${{ secrets.INFRACOST_API_KEY }}

      # Checkout the branch you want Infracost to compare costs against. This example is using the
      # target PR branch.
      - name: Checkout base branch
        uses: actions/checkout@v2
        with:
          ref: '${{ github.event.pull_request.base.ref }}'

      # Generate an Infracost output JSON from the comparison branch, so that Infracost can compare the cost difference.
      - name: Generate Infracost cost snapshot
        run: |
          infracost breakdown --path examples/terraform-directory/code \
                              --format json \
                              --out-file /tmp/prior.json

      - name: Checkout pr branch
        uses: actions/checkout@v2

      - name: Run Infracost
        run: |
          infracost diff --path examples/terraform-directory/code \
                              --format json \
                              --compare-to /tmp/prior.json \
                              --out-file /tmp/infracost.json

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
