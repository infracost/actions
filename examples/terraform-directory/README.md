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
          terraform_wrapper: false # This is required so the `terraform show` command outputs valid JSON

      # IMPORTANT: add any required steps here to setup cloud credentials so Terraform can run

      - name: Setup Infracost
        uses: infracost/actions/setup@v1
        with:
          api-key: ${{ secrets.INFRACOST_API_KEY }}
          
      - name: Run Infracost
        run: infracost breakdown --path=examples/terraform-directory/code --format=json --out-file=/tmp/infracost.json
        
      - name: Post the comment
        uses: infracost/actions/comment@v1
        with:
          path: /tmp/infracost.json
          behavior: update # Create a single comment and update it. The "quietest" option.
```
[//]: <> (END EXAMPLE)
