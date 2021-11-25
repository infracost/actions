# Multi-project

These examples show how to run Infracost actions against a multi-project setup using either an [Infracost config file](https://www.infracost.io/docs/multi_project/config_file) or a GitHub Actions build matrix.

## Using an Infracost config file

This example shows how to run Infracost actions with multiple Terraform projects using an [Infracost config file](https://www.infracost.io/docs/multi_project/config_file).

[//]: <> (BEGIN EXAMPLE)
```yml
name: Multi-project config file
on: [pull_request]

jobs:
  multi_project_config_file:
    name: Multi-project config file
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v2
      
      - name: Install Terraform
        uses: hashicorp/setup-terraform@v1
        with:
          terraform_wrapper: false # This is required so the `terraform show` command outputs valid JSON

      # IMPORTANT: add any required steps here to setup cloud credentials so Terraform can run

      - name: Setup Infracost
        uses: infracost/actions/setup@v1
        with:
          api-key: ${{ secrets.INFRACOST_API_KEY }}

      - name: Run Infracost
        run: infracost breakdown --config-file=examples/multi-project/code/infracost.yml --format=json --out-file=/tmp/infracost.json

      - name: Post the comment
        uses: infracost/actions/comment@v1
        with:
          path: /tmp/infracost.json
          behavior: update # Create a single comment and update it. The "quietest" option.
```
[//]: <> (END EXAMPLE)

## Using GitHub Actions build matrix 

This example shows how to run Infracost actions with multiple Terraform projects using a GitHub Actions build matrix. The first job uses a build matrix to generate multiple Infracost output JSON files and upload them as artifacts. The second job downloads these JSON files, combines them using `infracost output`, and posts a comment.

[//]: <> (BEGIN EXAMPLE)
```yml
name: Multi-project matrix
on: [pull_request]

jobs:
  multi_project_matrix:
    name: Multi-project matrix
    runs-on: ubuntu-latest

    strategy:
      matrix:
        dir: [dev, prod]

    steps:
      - uses: actions/checkout@v2
      
      - name: Install Terraform
        uses: hashicorp/setup-terraform@v1
        with:
          terraform_wrapper: false # This is required so the `terraform show` command outputs valid JSON

      # IMPORTANT: add any required steps here to setup cloud credentials so Terraform can run

      - name: Setup Infracost
        uses: infracost/actions/setup@v1
        with:
          api-key: ${{ secrets.INFRACOST_API_KEY }}
          
      - name: Run Infracost
        run: infracost breakdown --path=examples/multi-project/code/${{ matrix.dir }} --format=json --out-file=/tmp/infracost_${{ matrix.dir }}.json
        
      - name: Upload Infracost breakdown
        uses: actions/upload-artifact@v2
        with:
          name: infracost_jsons
          path: /tmp/infracost_${{ matrix.dir }}.json

  multi_project_matrix_merge:
    name: Multi-project matrix merge
    runs-on: ubuntu-latest
    needs: [multi_project_matrix]

    steps:
      - uses: actions/checkout@v2

      - name: Download all Infracost breakdown files
        uses: actions/download-artifact@v2
        with:
          path: /tmp
        
      - name: Setup Infracost
        uses: infracost/actions/setup@v1
        with:
          api-key: ${{ secrets.INFRACOST_API_KEY }}
          
      - name: Combine the results
        run: |
          infracost output --path="/tmp/infracost_jsons/*.json" --format=json --out-file=/tmp/infracost_combined.json
          
      - name: Post the comment
        uses: infracost/actions/comment@v1
        with:
          path: /tmp/infracost_combined.json
          behavior: update # Create a single comment and update it. The "quietest" option.
```
[//]: <> (END EXAMPLE)
