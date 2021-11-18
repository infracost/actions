# Infracost Setup Action

This GitHub Action downloads and installs the [Infracost CLI](https://github.com/infracost/infracost) in your GitHub Actions workflow. Subsequent steps in the same job can run the CLI in the same way it is run on the command line.

## Inputs

The action supports the following inputs:

- `api_key`: Required. The Infracost API key.

- `version`: Optional. [Version](https://github.com/infracost/infracost/releases) of Infracost CLI to install, e.g. 0.9.x or 0.9.13.

- `currency`: Optional. Convert output from USD to your preferred [ISO 4217 currency](https://en.wikipedia.org/wiki/ISO_4217#Active_codes), e.g. EUR, BRL or INR.

- `pricing_api_endpoint`: Optional. For [self-hosted](https://www.infracost.io/docs/cloud_pricing_api/self_hosted) users, endpoint of the Cloud Pricing API, e.g. https://cloud-pricing-api.

## Outputs

This action does not set any direct outputs.
