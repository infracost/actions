# Multi-Terraform workspace with matrix jobs

This example shows how to run Infracost in GitHub Actions against a Terraform project that uses multiple workspaces using parallel matrix jobs. The first job uses a matrix to generate the plan JSONs and the second job uses another matrix to generate multiple Infracost output JSON files. `infracost comment` command in the last job combines the Infracost JSON files and posts a single comment.

[//]: <> (BEGIN EXAMPLE)
```yml
name: Multi-workspace matrix
on: [pull_request]

jobs:
  multi-workspace-matrix:
    name: Multi-workspace matrix
    runs-on: ubuntu-latest
    env:
      TF_ROOT: examples/plan-json/multi-workspace-matrix/code

    strategy:
      matrix:
        include:
          # IMPORTANT: add any required secrets to setup cloud credentials so Terraform can run
          - tf_workspace: dev
            # GitHub actions doesn't support secrets in matrix values, so we use the name of the secret instead
            aws_access_key_id_secret: EXAMPLE_DEV_AWS_ACCESS_KEY_ID
            aws_secret_access_key_secret: EXAMPLE_DEV_AWS_SECRET_ACCESS_KEY
          - tf_workspace: prod
            aws_access_key_id_secret: EXAMPLE_PROD_AWS_ACCESS_KEY_ID
            aws_secret_access_key_secret: EXAMPLE_PROD_AWS_SECRET_ACCESS_KEY

    steps:
      - name: Checkout PR branch
        uses: actions/checkout@v2

      - name: Install Terraform
        uses: hashicorp/setup-terraform@v2
        with:
          terraform_wrapper: false # This is recommended so the `terraform show` command outputs valid JSON

      - name: Terraform init
        run: terraform init
        working-directory: ${{ env.TF_ROOT }}

      - name: Generate plan JSON
        run: |
          terraform plan -out=${{ matrix.tf_workspace }}-plan.cache -var-file=${{ matrix.tf_workspace }}.tfvars
          terraform show -json ${{ matrix.tf_workspace }}-plan.cache > ${{ matrix.tf_workspace }}-plan.json
        env:
          TF_WORKSPACE: ${{ matrix.tf_workspace }}
        working-directory: ${{ env.TF_ROOT }}

      - name: Setup Infracost
        uses: infracost/actions/setup@v2
        # See https://github.com/infracost/actions/tree/master/setup for other inputs
        # If you can't use this action, see Docker images in https://infracost.io/cicd
        with:
          api-key: ${{ secrets.INFRACOST_API_KEY }}

      # Generate an Infracost diff and save it to a JSON file.
      - name: Generate Infracost diff
        run: infracost diff --path=${TF_ROOT}/${{ matrix.tf_workspace }}-plan.json --format=json --out-file=/tmp/infracost_${{ matrix.tf_workspace }}.json
        env:
          AWS_ACCESS_KEY_ID: ${{ secrets[matrix.aws_access_key_id_secret] }}
          AWS_SECRET_ACCESS_KEY: ${{ secrets[matrix.aws_secret_access_key_secret] }}

      - name: Upload Infracost breakdown
        uses: actions/upload-artifact@v2
        with:
          name: infracost_workspace_jsons
          path: /tmp/infracost_${{ matrix.tf_workspace }}.json

  multi-workspace-matrix-merge:
    name: Multi-workspace matrix merge
    runs-on: ubuntu-latest
    needs: [multi-workspace-matrix]

    steps:
      - uses: actions/checkout@v2

      - name: Download all Infracost breakdown files
        uses: actions/download-artifact@v2
        with:
          name: infracost_workspace_jsons
          path: /tmp

      - name: Setup Infracost
        uses: infracost/actions/setup@v2
        with:
          api-key: ${{ secrets.INFRACOST_API_KEY }}

      # Posts a comment to the PR using the 'update' behavior.
      # This creates a single comment and updates it. The "quietest" option.
      # The other valid behaviors are:
      #   delete-and-new - Delete previous comments and create a new one.
      #   hide-and-new - Minimize previous comments and create a new one.
      #   new - Create a new cost estimate comment on every push.
      # See https://www.infracost.io/docs/features/cli_commands/#comment-on-pull-requests for other options.
      # The INFRACOST_ENABLE_CLOUD​=true section instructs the CLI to send its JSON output to Infracost Cloud.
      #   This SaaS product gives you visibility across all changes in a dashboard. The JSON output does not
      #   contain any cloud credentials or secrets.
      - name: Post Infracost comment
        run: |
          INFRACOST_ENABLE_CLOUD​=true infracost comment github --path="/tmp/infracost_*.json" \
                                   --repo=$GITHUB_REPOSITORY \
                                   --github-token=${{github.token}} \
                                   --pull-request=${{github.event.pull_request.number}} \
                                   --behavior=update
```
[//]: <> (END EXAMPLE)
