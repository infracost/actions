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

      - name: Terraform init
        run: terraform init
        working-directory: examples/terraform-directory/code

      - name: Terraform plan
        run: terraform plan -out tfplan.binary
        working-directory: examples/terraform-directory/code

      - name: Terraform show
        run: terraform show -json tfplan.binary > plan.json
        working-directory: examples/terraform-directory/code

      - name: Setup Infracost
        uses: infracost/actions/setup@v1
        with:
          api_key: ${{ secrets.INFRACOST_API_KEY }}
          
      - name: Run Infracost
        run: infracost breakdown --path=examples/terraform-directory/code --format=json --out-file=/tmp/infracost_breakdown.json
        
      - name: Post the comment
        uses: infracost/actions/comment@v1
        with:
          path: /tmp/infracost_breakdown.json
```
[//]: <> (END EXAMPLE)
