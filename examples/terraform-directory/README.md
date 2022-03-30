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
      - uses: actions/checkout@v2

      - name: Install terraform
        uses: hashicorp/setup-terraform@v1
        with:
          terraform_wrapper: false # This is recommended so the `terraform show` command outputs valid JSON

      # IMPORTANT: add any required steps here to setup cloud credentials so Terraform can run

      - name: Setup Infracost
        uses: infracost/actions/setup@v1
        with:
          api-key: ${{ secrets.INFRACOST_API_KEY }}

      - name: Run Infracost
        run: infracost breakdown --path=examples/terraform-directory/code --format=json --out-file=/tmp/infracost.json

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
