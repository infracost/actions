# Terragrunt

This example shows how to run Infracost actions with Terragrunt.

[//]: <> (BEGIN EXAMPLE)
```yml
name: Terragrunt
on: [pull_request]

jobs:
  terragrunt:
    name: Terragrunt
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v2
      
      - name: Install Terraform
        uses: hashicorp/setup-terraform@v1
        with:
          terraform_wrapper: false # This is recommended so the `terraform show` command outputs valid JSON

      - name: Setup Terragrunt
        uses: autero1/action-terragrunt@v1.1.0
        with:
          terragrunt_version: 0.35.9

      # IMPORTANT: add any required steps here to setup cloud credentials so Terraform/Terragrunt can run

      - name: Setup Infracost
        uses: infracost/actions/setup@v1
        with:
          api-key: ${{ secrets.INFRACOST_API_KEY }}
          
      - name: Run Infracost
        run: infracost breakdown --path=examples/terragrunt/code --format=json --out-file=/tmp/infracost.json
        
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
