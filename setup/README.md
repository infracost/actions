# Infracost setup action

This GitHub Action downloads and installs the [Infracost CLI](https://github.com/infracost/infracost) in your GitHub Actions workflow. Subsequent steps in the same job can run the CLI in the same way it is run on the command line using the [GitHub Actions `run` syntax](https://help.github.com/en/actions/reference/workflow-syntax-for-github-actions#jobsjob_idstepsrun).

## Usage

See the [top-level readme](https://github.com/infracost/actions) for examples of how this action can be used. You need to [Create a repo secret](https://docs.github.com/en/actions/configuring-and-managing-workflows/creating-and-storing-encrypted-secrets#creating-encrypted-secrets-for-a-repository) called `INFRACOST_API_KEY` with your API key and pass it into this action:

```yml
steps:
  - name: Setup Infracost
    uses: infracost/actions/setup@v2
    with:
      api-key: ${{ secrets.INFRACOST_API_KEY }}
```

## Inputs

The action supports the following inputs:

- `api-key`: Required. Your Infracost API key. You can get a free API key or retrieve your existing one from [Infracost Cloud](https://dashboard.infracost.io) > Org Settings.

- `version`: Optional, defaults to `0.10.x`. [SemVer ranges](https://www.npmjs.com/package/semver#ranges) are supported, so instead of a [full version](https://github.com/infracost/infracost/releases) string, you can use `0.10.x`. This enables you to automatically get the latest backward compatible changes in the 0.10 release (e.g. new resources or bug fixes).

- `currency`: Optional. Convert output from USD to your preferred [ISO 4217 currency](https://en.wikipedia.org/wiki/ISO_4217#Active_codes), e.g. EUR, BRL or INR.

- `pricing-api-endpoint`: Optional. For self-hosted users, endpoint of the Cloud Pricing API, e.g. https://cloud-pricing-api.

## Outputs

This action does not set any direct outputs.
