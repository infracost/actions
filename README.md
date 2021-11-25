# Infracost GitHub Actions

[Infracost](https://www.infracost.io/) enables you to see cloud cost estimates for Terraform in pull requests 💰 This project provides a set of GitHub Actions for Infracost:
- **[setup](setup)**: downloads and installs the Infracost CLI in your GitHub Actions workflow.
- **[comment](comment)**: adds comments to pull requests.
- **[get-comment](get-comment)**: reads a comment from a pull requests.

    <img src="https://raw.githubusercontent.com/infracost/infracost-gh-action/master/screenshot.png" width=480 alt="Example usage" />

## Usage

The following steps assume a simple Terraform directory is being used, we recommend you use a more relevant [example](examples) if required.

1. Retrieve your Infracost API key by running `infracost configure get api_key`. If you don't have one, [download Infracost](https://www.infracost.io/docs/#quick-start) and run `infracost register` to get a free API key.

2. [Create a repo secret](https://docs.github.com/en/actions/configuring-and-managing-workflows/creating-and-storing-encrypted-secrets#creating-encrypted-secrets-for-a-repository) called `INFRACOST_API_KEY` with your API key.

3. Create required repo secrets for any cloud credentials that are needed for Terraform to run. If you have multiple projects/workspaces, consider using an Infracost [config-file](https://www.infracost.io/docs/multi_project/config_file) to define the projects.

    - **Terraform Cloud/Enterprise users**: if you use Remote Execution Mode, you should follow [setup-terraform](https://github.com/hashicorp/setup-terraform) instructions to set the inputs `cli_config_credentials_token`, and `cli_config_credentials_hostname` for Terraform Enterprise.
    - **AWS users**: use [aws-actions/configure-aws-credentials](https://github.com/aws-actions/configure-aws-credentials), the [Terraform docs](https://registry.terraform.io/providers/hashicorp/aws/latest/docs#environment-variables) explain other options.
    - **Azure users**: the [Terraform docs](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/guides/service_principal_client_secret) explain the options. The [Azure/login](https://github.com/Azure/login) GitHub Actions might also be useful; we haven't tested these with Terraform.
    - **Google users**: the [Terraform docs](https://registry.terraform.io/providers/hashicorp/google/latest/docs/guides/provider_reference#full-reference) explain the options, e.g. using `GOOGLE_CREDENTIALS`.

4. Create a new file in `.github/workflows/infracost.yml` in your repo with the following content.

    ```yaml
    # The GitHub Actions docs (https://docs.github.com/en/actions/reference/workflow-syntax-for-github-actions#on)
    # describe other options for 'on', 'pull_request' is a good default.
    on: [pull_request]
    jobs:
      infracost:
        runs-on: ubuntu-latest
        name: Run Infracost
        steps:
          - name: Check out repository
            uses: actions/checkout@v2

          # Typically the Infracost actions will be used in conjunction with
          # https://github.com/hashicorp/setup-terraform. Subsequent steps in
          # can run Terraform commands as they would in the shell.
          - name: Install terraform
            uses: hashicorp/setup-terraform@v1
            with:
              terraform_wrapper: false # This is required so the `terraform show` command outputs valid JSON

          # IMPORTANT: add any required steps here to setup cloud credentials so Terraform can run

          - name: Terraform init
            run: terraform init
            working-directory: PATH/TO/MY_CODE

          - name: Terraform plan
            run: terraform plan -out tfplan.binary
            working-directory: PATH/TO/MY_CODE

          - name: Terraform show
            run: terraform show -json tfplan.binary > /tmp/plan.json
            working-directory: PATH/TO/MY_CODE

          # Install the Infracost CLI, see https://github.com/infracost/actions/tree/master/setup
          # for other inputs such as version, and pricing-api-endpoint (for self-hosted users).
          - name: Setup Infracost
            uses: infracost/actions/setup@v1
            with:
              api-key: ${{ secrets.INFRACOST_API_KEY }}

          # Generate Infracost JSON output, the following docs might be useful:
          # Multi-project/workspaces: https://www.infracost.io/docs/multi_project/config_file
          # Combine Infracost JSON files: https://www.infracost.io/docs/multi_project/report
          - name: Generate Infracost JSON
            run: infracost breakdown --path /tmp/plan.json --format json --out-file /tmp/infracost.json
            # Env vars can be set using the usual GitHub Actions syntax
            # env:
            #   MY_ENV: ${{ secrets.MY_ENV }}

          # See https://github.com/infracost/actions/tree/master/comment
          # for other inputs such as target-type.
          - name: Post Infracost comment
            uses: infracost/actions/comment@v1
            with:
              path: /tmp/infracost.json
              # Choose the commenting behavior, 'update' is a good default:
              behavior: update # Create a single comment and update it. The "quietest" option.                 
              # behavior: delete-and-new # Delete previous comments and create a new one.
              # behavior: hide-and-new # Minimize previous comments and create a new one.
              # behavior: new # Create a new cost estimate comment on every push.
    ```

4. Send a new pull request to change something in Terraform that costs money. You should see a pull request comment that gets updated, e.g. the 📉 and 📈 emojis will update as changes are pushed! Check the GitHub Actions logs and [this page](https://www.infracost.io/docs/integrations/cicd#cicd-troubleshooting) if there are issues.

## Examples

The [examples](examples) directory demonstrates how these actions can be used in different workflows, including:
  - [Terraform directory](examples/terraform-directory): a Terraform directory containing HCL code
  - [Terraform plan JSON](examples/terraform-plan-json): a Terraform plan JSON file
  - [Terragrunt](examples/terragrunt): a Terragrunt project
  - [Terraform Cloud/Enterprise](terraform-cloud-enterprise): a Terraform project using Terraform Cloud/Enterprise
  - [Multi-project using config file](examples/multi-project/README.md#using-an-infracost-config-file): multiple Terraform projects using the Infracost [config file](https://www.infracost.io/docs/multi_project/config_file)
  - [Multi-project using build matrix](examples/multi-project/README.md#using-github-actions-build-matrix): multiple Terraform projects using GitHub Actions build matrix
  - [Multi-Terraform workspace](examples/multi-terraform-workspace): multiple Terraform workspaces using the Infracost [config file](https://www.infracost.io/docs/multi_project/config_file)
  - [Thresholds](examples/thresholds): only post a comment when cost thresholds are exceeded
## Contributing

Issues and pull requests are welcome! For development details, see the [contributing](CONTRIBUTING.md) guide. For major changes, including interface changes, please open an issue first to discuss what you would like to change. [Join our community Slack channel](https://www.infracost.io/community-chat), we are a friendly bunch and happy to help you get started :)

## License

[Apache License 2.0](https://choosealicense.com/licenses/apache-2.0/)
