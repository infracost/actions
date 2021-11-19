# Terragrunt

[//]: <> (BEGIN EXAMPLE)
```yml
name: Terragrunt
on:
  push:
    branches:
      - master
  pull_request:

jobs:
  matrix:
    name: Matrix
    runs-on: ubuntu-latest

    strategy:
      matrix:
        dir: [dev, prod]

    steps:
      - uses: actions/checkout@v2
      
      - name: Install Terraform
        uses: hashicorp/setup-terraform@v1
        with:
          terraform_wrapper: false # This is required so that Terraform binary outputs valid JSON

      - name: Setup Terragrunt
        uses: autero1/action-terragrunt@v1.1.0
        with:
          terragrunt_version: 0.35.9

      - name: Setup Infracost
        uses: infracost/actions/setup@v1
        with:
          api_key: ${{ secrets.INFRACOST_API_KEY }}
          
      - name: Run Infracost
        run: infracost breakdown --path=examples/matrix/code/${{ matrix.dir }} --format=json --out-file=/tmp/infracost_breakdown_${{ matrix.dir }}.json
        
      - name: Upload Infracost breakdown
        uses: actions/upload-artifact@v2
        with:
          name: infracost_breakdown_${{ matrix.dir }}.json
          path: infracost_breakdown_${{ matrix.dir }}.json
    
  matrix_merge:
    name: Matrix merge
    runs-on: ubuntu-latest
    needs: [matrix]

    steps:
      - name: Download all Infracost breakdown files
        uses: actions/download-artifact@v2
        
      - name: ls
        run: ls
```
[//]: <> (END EXAMPLE)
