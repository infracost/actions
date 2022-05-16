# Examples

All examples post a single comment on merge requests, which gets updated as more changes are pushed.

## Terraform project examples

These examples work by using the default Infracost CLI option that parses HCL, thus a Terraform Plan JSON is not needed.

  - [Terraform/Terragrunt projects (single or multi)](terraform-project): a repository containing one or more (e.g. mono repos) Terraform or Terragrunt projects
  - [Multi-projects using a config file](multi-project-config-file): repository containing multiple Terraform projects that need different inputs, i.e. variable files or Terraform workspaces
  - [Private Terraform module](private-terraform-module): a Terraform project using a private Terraform module
  - [Slack](slack): send cost estimates to Slack

## Plan JSON examples

These examples are for advanced use cases where the estimate is generated from Terraform plan JSON files.

- [Terragrunt](plan-json/terragrunt): Generate plan JSONs for a Terragrunt project
- [Terraform Cloud/Enterprise](plan-json/terraform-cloud-enterprise): Generate a plan JSON for a Terraform project using Terraform Cloud/Enterprise
- [Multi-project using parallel matrix jobs](plan-json/multi-project-matrix): Generate multiple plan JSONs for different Terraform projects using parallel matrix jobs
- [Multi-Terraform workspaces using parallel matrix jobs](plan-json/multi-workspace-matrix): Generate multiple plan JSONs for different Terraform workspaces using parallel matrix jobs

## Cost policy examples

- [OPA](https://www.infracost.io/docs/features/cost_policies/): check Infracost cost estimates against policies using Open Policy Agent
- [Sentinel](sentinel): check Infracost cost estimates against policies using Hashicorp's Sentinel

See the [contributing](../CONTRIBUTING.md) guide if you'd like to add an example.
