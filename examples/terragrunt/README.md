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
        
      - name: Post the comment
        uses: infracost/actions/comment@v1
        with:
          path: /tmp/infracost.json
          behavior: update # Create a single comment and update it. See https://github.com/infracost/actions/tree/master/comment for other options
```
[//]: <> (END EXAMPLE)
