# Multi-Terraform workspace

This example shows how to run Infracost actions against a Terraform project that uses multiple workspaces using an [Infracost config file](https://www.infracost.io/docs/multi_project/config_file).

[//]: <> (BEGIN EXAMPLE)
```yml
name: Multi-terraform workspace config file
on: [pull_request]

jobs:
  multi_terraform_workspace_config_file:
    name: Multi-Terraform workspace config file
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v2
      
      - name: Install Terraform
        uses: hashicorp/setup-terraform@v1
        with:
          terraform_wrapper: false # This is required so the `terraform show` command outputs valid JSON

      - name: Setup Infracost
        uses: infracost/actions/setup@v1
        with:
          api-key: ${{ secrets.INFRACOST_API_KEY }}

      - name: Run Infracost
        run: infracost breakdown --config-file=examples/multi-terraform-workspace/code/infracost.yml --format=json --out-file=/tmp/infracost.json

      - name: Post the comment
        uses: infracost/actions/comment@v1
        with:
          path: /tmp/infracost.json
```
[//]: <> (END EXAMPLE)
