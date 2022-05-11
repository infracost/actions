# Terraform/Terragrunt project (single or multi)

This example shows how to run Infracost in GitHub Actions with multiple Terraform/Terragrunt projects, both single projects or mono-repos that contain multiple projects.

[//]: <> (BEGIN EXAMPLE)
```yml
name: Terraform project
on: [pull_request]

jobs:
  terraform-project:
    name: Terraform project
    runs-on: ubuntu-latest
    env:
      TF_ROOT: examples/terraform-project/code

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
          infracost breakdown --path ${TF_ROOT} \
                              --format json \
                              --out-file /tmp/infracost-base.json

      - name: Checkout PR branch
        uses: actions/checkout@v2

      - name: Run Infracost
        run: |
          infracost diff --path ${TF_ROOT} \
                              --format json \
                              --compare-to /tmp/infracost-base.json \
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
