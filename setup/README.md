# Infracost setup action

This GitHub Action downloads and installs the [Infracost CLI](https://github.com/infracost/infracost) in your GitHub Actions workflow. Subsequent steps in the same job can run the CLI in the same way it is run on the command line using the [GitHub Actions `run` syntax](https://help.github.com/en/actions/reference/workflow-syntax-for-github-actions#jobsjob_idstepsrun).

## Usage

The action can be used as follows:

```yml
steps:
  - name: Setup Infracost
    uses: infracost/actions/setup@v1
    with:
      api-key: ${{ secrets.INFRACOST_API_KEY }}
```

## Inputs

The action supports the following inputs:

- `api-key`: Required. Your Infracost API key. It can be retrieved by running `infracost configure get api_key`. If you don't have one, [download Infracost](https://www.infracost.io/docs/#quick-start) and run `infracost register` to get a free API key.

- `version`: Optional, defaults to `0.9.x`. [Semver Ranges](https://www.npmjs.com/package/semver#ranges) are supported, so instead of a [full version](https://github.com/infracost/infracost/releases) string, you can use `0.9.x`. This enables you to automatically get the latest backward compatible changes in the 0.9 release (e.g. new resources or bug fixes).

- `currency`: Optional. Convert output from USD to your preferred [ISO 4217 currency](https://en.wikipedia.org/wiki/ISO_4217#Active_codes), e.g. EUR, BRL or INR.

- `pricing-api-endpoint`: Optional. For [self-hosted](https://www.infracost.io/docs/cloud_pricing_api/self_hosted) users, endpoint of the Cloud Pricing API, e.g. https://cloud-pricing-api.

## Outputs

This action does not set any direct outputs.
