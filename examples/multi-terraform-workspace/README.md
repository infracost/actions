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
      - uses: actions/checkout@v2
      
      - name: Install Terraform
        uses: hashicorp/setup-terraform@v1
        with:
          terraform_wrapper: false # This is recommended so the `terraform show` command outputs valid JSON

      - name: Setup Infracost
        uses: infracost/actions/setup@v1
        with:
          api-key: ${{ secrets.INFRACOST_API_KEY }}

      - name: Run Infracost
        run: infracost breakdown --config-file=examples/multi-terraform-workspace/code/infracost.yml --format=json --out-file=/tmp/infracost.json
        env:
          # IMPORTANT: add any required secrets to setup cloud credentials so Terraform can run
          DEV_AWS_ACCESS_KEY_ID: ${{ secrets.EXAMPLE_DEV_AWS_ACCESS_KEY_ID }}
          DEV_AWS_SECRET_ACCESS_KEY: ${{ secrets.EXAMPLE_DEV_AWS_SECRET_ACCESS_KEY }}
          PROD_AWS_ACCESS_KEY_ID: ${{ secrets.EXAMPLE_PROD_AWS_ACCESS_KEY_ID }}
          PROD_AWS_SECRET_ACCESS_KEY: ${{ secrets.EXAMPLE_PROD_AWS_SECRET_ACCESS_KEY }}

      - name: Post Infracost comment
        run: |

          # Posts a comment to the PR using the 'update' behavior.
          # This creates a single comment and updates it. The "quietest" option.
          # The other valid behaviors are:
          #   delete-and-new - Delete previous comments and create a new one.
          #   hide-and-new - Minimize previous comments and create a new one.
          #   new - Create a new cost estimate comment on every push.
          #
          # See https://www.infracost.io/docs/features/cli_commands/#comment-on-pull-requests
          # for other inputs such as target-type.

          infracost comment github --path /tmp/infracost.json \
                                   --repo $GITHUB_REPOSITORY \
                                   --github-token ${{github.token}} \
                                   --pull-request ${{github.event.pull_request.number}} \
                                   --behavior update
```
[//]: <> (END EXAMPLE)
