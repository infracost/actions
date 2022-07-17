# Terraform Cloud/Enterprise

This example shows how to run Infracost on GitHub CI with Terraform Cloud and Terraform Enterprise. It assumes you have set a GitHub secret for the Terraform Cloud token (`TFC_TOKEN`), which is used to run a speculative plan and fetch the plan JSON from Terraform Cloud.

[//]: <> (BEGIN EXAMPLE)
```yml
name: Terraform Cloud/Enterprise
on: [pull_request]

jobs:
  terraform-cloud-enterprise:
    name: Terraform Cloud/Enterprise
    runs-on: ubuntu-latest
    env:
      TF_ROOT: examples/plan-json/terraform-cloud-enterprise/code
      TFC_HOST: app.terraform.io # Change this if you're using Terraform Enterprise

    steps:
      - name: Checkout PR branch
        uses: actions/checkout@v2

      - name: Setup Terraform
        uses: hashicorp/setup-terraform@v2
        with:
          cli_config_credentials_token: ${{ secrets.TFC_TOKEN }}
          cli_config_credentials_hostname: ${{ env.TFC_HOST }}
          terraform_wrapper: false # This is recommended so the `terraform show` command outputs valid JSON

      # IMPORTANT: add any required steps here to setup cloud credentials so Terraform can run
      - name: Terraform init
        run: terraform init
        working-directory: ${{ env.TF_ROOT }}

      # When using TFC remote execution, terraform doesn't allow us to save the plan output.
      # So we have to save the plan logs so we can parse out the run ID and fetch the plan JSON
      - name: Retrieve plan JSONs
        run: |
          echo "Running terraform plan"
          terraform plan -no-color | tee /tmp/plan_logs.txt

          echo "Parsing the run URL and ID from the logs"
          run_url=$(grep -A1 'To view this run' /tmp/plan_logs.txt | tail -n 1)
          run_id=$(basename $run_url)

          echo "Getting the run plan response from https://$TFC_HOST/api/v2/runs/$run_id/plan"
          run_plan_resp=$(wget -q -O - --header="Authorization: Bearer ${{ secrets.TFC_TOKEN }}" "https://$TFC_HOST/api/v2/runs/$run_id/plan")
          echo "Extracting the plan JSON path"
          plan_json_path=$(echo $run_plan_resp | sed 's/.*\"json-output\":\"\([^\"]*\)\".*/\1/')

          echo "Downloading the plan JSON from https://$TFC_HOST$plan_json_path"
          wget -q -O plan.json --header="Authorization: Bearer ${{ secrets.TFC_TOKEN }}" "https://$TFC_HOST$plan_json_path"
        working-directory: ${{ env.TF_ROOT }}

      - name: Setup Infracost
        uses: infracost/actions/setup@v2
        # See https://github.com/infracost/actions/tree/master/setup for other inputs
        # If you can't use this action, see Docker images in https://infracost.io/cicd
        with:
          api-key: ${{ secrets.INFRACOST_API_KEY }}

      # Generate an Infracost diff and save it to a JSON file.
      - name: Generate Infracost diff
        run: |
          infracost diff --path=${TF_ROOT}/plan.json \
                         --format=json \
                         --out-file=/tmp/infracost.json

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
          INFRACOST_ENABLE_CLOUD​=true infracost comment github --path=/tmp/infracost.json \
                                   --repo=$GITHUB_REPOSITORY \
                                   --github-token=${{github.token}} \
                                   --pull-request=${{github.event.pull_request.number}} \
                                   --behavior=update
```
[//]: <> (END EXAMPLE)
