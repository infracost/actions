# Terraform cloud

[//]: <> (BEGIN EXAMPLE)
```yml
name: Terraform cloud
on: [pull_request]

jobs:
  terraform-cloud:
    name: Terraform cloud
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v2
      
      - name: Install terraform
        uses: hashicorp/setup-terraform@v1
        with:
          terraform_wrapper: false # This is required so the `terraform show` command outputs valid JSON
          cli_config_credentials_token: $${{ secrets.TFC_TOKEN }}

      - name: Setup Infracost
        uses: infracost/actions/setup@v1
        with:
          api-key: ${{ secrets.INFRACOST_API_KEY }}
          
      - name: Run Infracost
        run: infracost breakdown --path=examples/terraform-cloud/code --format=json --out-file=/tmp/infracost.json
        env:
          INFRACOST_TERRAFORM_CLOUD_TOKEN: ${{ secrets.TFC_TOKEN }} # TODO: can be removed once https://github.com/infracost/infracost/pull/1148 is released
        
      - name: Post the comment
        uses: infracost/actions/comment@v1
        with:
          path: /tmp/infracost.json
```
[//]: <> (END EXAMPLE)
