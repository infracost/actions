# Terraform Cloud/Enterprise

This example shows how to run Infracost actions with Terraform Cloud and Terraform Enterprise. It assumes you have set a GitHub repo secret for the Terraform Cloud token (`TFC_TOKEN`). This token is used by the Infracost CLI run a speculative plan and fetch the plan JSON from Terraform Cloud to generate the cost estimate comment.

In the future, we'll add an example of how you can trigger the Infracost actions from Terraform Cloud's GitHub status checks.

[//]: <> (BEGIN EXAMPLE)
```yml
name: Terraform Cloud/Enterprise
on: [pull_request]

jobs:
  terraform-cloud-enterprise:
    name: Terraform Cloud/Enterprise
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

      - name: Run Infracost
        run: |
          infracost breakdown --path=examples/terraform-cloud-enterprise/code \
                              --format=json \
                              --out-file=/tmp/prior.json
        env:
          # Terraform cloud credentials used to fetch Terraform vars.
          INFRACOST_TERRAFORM_CLOUD_TOKEN: ${{ secrets.TFC_TOKEN }}
          # INFRACOST_TERRAFORM_CLOUD_HOST: my-tfe-host.com # For Terraform Enterprise users only.

      - name: Checkout pr branch
        uses: actions/checkout@v2

      - name: Run Infracost
        run: |
          infracost diff --path=examples/terraform-cloud-enterprise/code \
                              --format=json \
                              --compare-to=/tmp/prior.json \
                              --out-file=/tmp/infracost.json
        env:
          # Terraform cloud credentials used to fetch Terraform vars.
          INFRACOST_TERRAFORM_CLOUD_TOKEN: ${{ secrets.TFC_TOKEN }}
          # INFRACOST_TERRAFORM_CLOUD_HOST: my-tfe-host.com # For Terraform Enterprise users only.

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
