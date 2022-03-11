# Terraform plan JSON

This example shows how to run Infracost actions with a Terraform plan JSON file. Installing Terraform is not needed since the Infracost CLI can use the plan JSON directly.

[//]: <> (BEGIN EXAMPLE)
```yml
name: Terraform plan JSON
on: [pull_request]

jobs:
  terraform-plan-json:
    name: Terraform plan JSON
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v2
      
      - name: Setup Infracost
        uses: infracost/actions/setup@v1
        with:
          api-key: ${{ secrets.INFRACOST_API_KEY }}
          
      - name: Run Infracost
        run: infracost breakdown --path=examples/terraform-plan-json/code/plan.json --format=json --out-file=/tmp/infracost.json
        
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
