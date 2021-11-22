# Infracost GitHub Actions

[Infracost](https://www.infracost.io/) enables you to see cloud cost estimates for Terraform in pull requests. This project provides a set of GitHub Actions for Infracost:
- [setup](setup): downloads and installs the Infracost CLI in your GitHub Actions workflow.
- [comment](comment): coming soon! Adds comments to pull requests.

The [examples](examples) directory demonstrates how these actions can be used in different workflows, including how it can be used with:
  - [Terraform directory](examples/terraform-directory): a Terraform directory containing HCL code
  - [Terraform plan JSON](terraform-plan-json): a Terraform plan JSON file
  - [Terragrunt](examples/terragrunt): a Terragrunt project

## Usage

Typically these actions will be used in conjunction with the [setup-terraform](https://github.com/hashicorp/setup-terraform) action. Subsequent steps in the same job can run arbitrary Infracost or Terraform commands using the [GitHub Actions `run` syntax](https://help.github.com/en/actions/reference/workflow-syntax-for-github-actions#jobsjob_idstepsrun). This allows most commands to work exactly like they do on your local command line.

Assuming you [downloaded Infracost](https://www.infracost.io/docs/#quick-start) and ran `infracost register` to get an API key, you should:

1. [Add repo secrets](https://docs.github.com/en/actions/configuring-and-managing-workflows/creating-and-storing-encrypted-secrets#creating-encrypted-secrets-for-a-repository) for `INFRACOST_API_KEY` and any other required credentials to your GitHub repo (e.g. `AWS_ACCESS_KEY_ID`).

2. Install the Infracost CLI:

    ```yml
    steps:
    - uses: infracost/actions/setup@master
      with:
        api_key: ${{ secrets.INFRACOST_API_KEY }}
    ```

    An optional `version` input is available, it supports [Semver Ranges](https://www.npmjs.com/package/semver#ranges), so instead of a [full version](https://github.com/infracost/infracost/releases) string, you can use `0.9.x` (default). This enables you to automatically get the latest backward compatible changes in the 0.9 release (e.g. new resources or bug fixes).

    See the [setup](setup) action readme for other options such as `currency` and `pricing_api_endpoint` (for [self-hosted](https://www.infracost.io/docs/cloud_pricing_api/self_hosted) users).

3. Create a new file in `.github/workflows/infracost.yml` in your repo with the following content. The GitHub Actions [docs](https://docs.github.com/en/actions/reference/workflow-syntax-for-github-actions#on) describe other options for `on`, though `pull_request` is what we recommend as a good default.

    ```yaml
    on: [pull_request]
    jobs:
      infracost:
        runs-on: ubuntu-latest
        name: Run Infracost
        steps:
          - name: Check out repository
            uses: actions/checkout@v2

          - name: Install terraform
            uses: hashicorp/setup-terraform@v1
            with:
              terraform_wrapper: false

          - name: Terraform init
            run: terraform init
            working-directory: path/to/my-terraform

          - name: Terraform plan
            run: terraform plan -out plan.tfplan
            working-directory: path/to/my-terraform

          - name: Terraform show
            id: tf_show
            run: terraform show -json plan.tfplan
            working-directory: path/to/my-terraform

          - name: Save Terraform Plan JSON
            run: echo '${{ steps.tf_show.outputs.stdout }}' > plan.json # Do not change

          - name: Setup Infracost
            uses: infracost/actions/setup@master
            with:
              api_key: ${{ secrets.INFRACOST_API_KEY }}

          - name: Infracost breakdown
            run: infracost breakdown --path plan.json --format json --out-file infracost.json

          - name: Infracost output
            run: infracost output --path infracost.json --format github-comment --out-file infracost-comment.md

          - name: Post comment
            uses: marocchino/sticky-pull-request-comment@v2
            with:
              path: infracost-comment.md
    ```

    You might find the following Infracost CLI [docs](https://www.infracost.io/docs/ ) pages useful:
    - [`infracost output`](https://www.infracost.io/docs/multi_project/report) is used to combine and output Infracost JSON files in different formats.
    - [config file](https://www.infracost.io/docs/multi_project/config_file)

4. Send a new pull request to change something in Terraform that costs money; a comment should be posted on the pull request. Check the GitHub Actions logs and [this page](https://www.infracost.io/docs/integrations/cicd#cicd-troubleshooting) if there are issues.

## Contributing

Issues and pull requests are welcome! For development details, see the [contributing](CONTRIBUTING.md) guide. For major changes, including interface changes, please open an issue first to discuss what you would like to change. [Join our community Slack channel](https://www.infracost.io/community-chat), we are a friendly bunch and happy to help you get started :)

## License

[Apache License 2.0](https://choosealicense.com/licenses/apache-2.0/)
