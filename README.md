# Infracost GitHub Actions

This project provides a set of GitHub Actions for Infracost, so you can see cloud cost estimates for Terraform in pull requests 💰 

<img src=".github/assets/screenshot.png" alt="Example screenshot" />

## Quick start

The following steps assume a simple Terraform directory is being used, we recommend you use a more relevant [example](#examples) if required.

1. Retrieve your Infracost API key by running `infracost configure get api_key`. We recommend using your same API key in all environments. If you don't have one, [download Infracost](https://www.infracost.io/docs/#quick-start) and run `infracost register` to get a free API key.

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
        runs-on: ubuntu-latest # The following are JavaScript actions (not Docker)
        env:
          working-directory: PATH/TO/TERRAFORM/CODE # Update this!

        name: Run Infracost
        steps:
          - name: Check out repository
            uses: actions/checkout@v2

          # Typically the Infracost actions will be used in conjunction with
          # https://github.com/hashicorp/setup-terraform. Subsequent steps
          # can run Terraform commands as they would in the shell.
          - name: Install terraform
            uses: hashicorp/setup-terraform@v1
            with:
              terraform_wrapper: false # This is recommended so the `terraform show` command outputs valid JSON

          # IMPORTANT: add any required steps here to setup cloud credentials so Terraform can run

          - name: Terraform init
            run: terraform init
            working-directory: ${{ env.working-directory }}

          - name: Terraform plan
            run: terraform plan -out tfplan.binary
            working-directory: ${{ env.working-directory }}

          - name: Terraform show
            run: terraform show -json tfplan.binary > plan.json
            working-directory: ${{ env.working-directory }}

          # Install the Infracost CLI, see https://github.com/infracost/actions/tree/master/setup
          # for other inputs such as version, and pricing-api-endpoint (for self-hosted users).
          - name: Setup Infracost
            uses: infracost/actions/setup@v1
            with:
              api-key: ${{ secrets.INFRACOST_API_KEY }}

          # Generate Infracost JSON output, the following docs might be useful:
          # Multi-project/workspaces: https://www.infracost.io/docs/features/config_file
          # Combine Infracost JSON files: https://www.infracost.io/docs/features/cli_commands/#combined-output-formats
          - name: Generate Infracost JSON
            run: infracost breakdown --path plan.json --format json --out-file /tmp/infracost.json
            working-directory: ${{ env.working-directory }}
            # Env vars can be set using the usual GitHub Actions syntax
            # See the list of supported Infracost env vars here: https://www.infracost.io/docs/integrations/environment_variables/
            # env:
            #   MY_ENV: ${{ secrets.MY_ENV }}

          # See https://www.infracost.io/docs/features/cli_commands/#comment-on-pull-requests
          # for other inputs such as target-type.
          - name: Post Infracost comment
            run: |

              # Posts a comment to the PR using the 'update' behavior.
              # This creates a single comment and updates it. The "quietest" option.
              # The other valid behaviors are:
              #   delete-and-new - Delete previous comments and create a new one.
              #   hide-and-new - Minimize previous comments and create a new one.
              #   new - Create a new cost estimate comment on every push.

              infracost comment github --path /tmp/infracost.json \
                                       --repo $GITHUB_REPOSITORY \
                                       --github-token ${{github.token}} \
                                       --pull-request ${{github.event.pull_request.number}} \
                                       --behavior update
    ```

4. 🎉 That's it! Send a new pull request to change something in Terraform that costs money. You should see a pull request comment that gets updated, e.g. the 📉 and 📈 emojis will update as changes are pushed!

    If there are issues, check the GitHub Actions logs and [this page](https://www.infracost.io/docs/troubleshooting/).

## Examples

The [examples](examples) directory demonstrates how these actions can be used in different workflows, including:
  - [Terraform directory](examples/terraform-directory): a Terraform directory containing HCL code
  - [Terraform plan JSON](examples/terraform-plan-json): a Terraform plan JSON file
  - [Terragrunt](examples/terragrunt): a Terragrunt project
  - [Terraform Cloud/Enterprise](examples/terraform-cloud-enterprise): a Terraform project using Terraform Cloud/Enterprise
  - [Multi-project using config file](examples/multi-project/README.md#using-an-infracost-config-file): multiple Terraform projects using the Infracost [config file](https://www.infracost.io/docs/multi_project/config_file)
  - [Multi-project using build matrix](examples/multi-project/README.md#using-github-actions-build-matrix): multiple Terraform projects using GitHub Actions build matrix
  - [Multi-Terraform workspace](examples/multi-terraform-workspace): multiple Terraform workspaces using the Infracost [config file](https://www.infracost.io/docs/multi_project/config_file)
  - [Private Terraform module](examples/private-terraform-module): a Terraform project using a private Terraform module
  - [Slack](examples/slack): send cost estimates to Slack

### Cost policies

Infracost policies enable centralized teams, who are often helping others with cloud costs, to provide advice before resources are launched, setup guardrails, and prevent human error. Follow [our docs](https://www.infracost.io/docs/features/cost_policies/) to use Infracost's native support for Open Policy Agent (OPA) policies. This enables you to see passing/failing policies in Infracost pull request comments (shown below) without having to install anything else.

![](.github/assets/policy-passing-github.png)

If you use HashiCorp Sentinel, follow [our example](examples/sentinel) to output the policy pass/fail results into CI/CD logs.

## Actions

We recommend you use the above [quick start](#quick-start) guide and examples, which combine the following individual actions:
- [setup](setup): downloads and installs the Infracost CLI in your GitHub Actions workflow.

### Deprecated Actions
- [comment](comment): adds comments to pull requests. This action is deprecated, please use `infracost comment` directly.
- [get-comment](get-comment): reads a comment from a pull request. This action is deprecated.

## Contributing

Issues and pull requests are welcome! For development details, see the [contributing](CONTRIBUTING.md) guide. For major changes, including interface changes, please open an issue first to discuss what you would like to change. [Join our community Slack channel](https://www.infracost.io/community-chat), we are a friendly bunch and happy to help you get started :)

## License

[Apache License 2.0](https://choosealicense.com/licenses/apache-2.0/)
