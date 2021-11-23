# Infracost Comment Action

Infracost enables you to see cloud cost estimates for Terraform in pull requests.

This GitHub Action takes infracost "breakdown" json output and posts it as a GitHub comment. It assumes the infracost binary has already been installed using `infracost/actions/setup`.

## Usage

Assuming you have [downloaded Infracost](https://www.infracost.io/docs/#quick-start) and run `infracost register` to get an API key, you should:

1. [Add repo secrets](https://docs.github.com/en/actions/configuring-and-managing-workflows/creating-and-storing-encrypted-secrets#creating-encrypted-secrets-for-a-repository) for `INFRACOST_API_KEY` and any other required credentials to your GitHub repo (e.g. `AWS_ACCESS_KEY_ID`).

2. Install the Infracost CLI; this action uses the `infracost output` command to generate the comment markdown from a terraform plan file.

    ```yml
    steps:
    - uses: infracost/actions/setup@master
      with:
        api_key: ${{ secrets.INFRACOST_API_KEY }}
    ```

3. Create a new file in `.github/workflows/infracost.yml` in your repo with the following content. Typically this action will be use the [setup-terraform](https://github.com/hashicorp/setup-terraform) action to generate a plan.json file.

```yaml
on: [pull_request]
jobs:
  infracost:
    runs-on: ubuntu-latest
    name: Post Infracost comment
    steps:
      - name: Check out repository
        uses: actions/checkout@v2

      - name: Install terraform
        uses: hashicorp/setup-terraform@v1

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

      - name: Infracost comment
        uses: infracost/actions/comment@master
        with: 
          path: infracost.json
```

## Inputs

The action supports the following inputs:

- `path`: Required. The path to the infracost breakdown json that will be passed to infracost output. For multiple paths, pass a glob pattern or a JSON array of paths.

- `behavior`: Optional, defaults to `update`. The behavior to use when posting cost estimate comments. Must be one of the following:  
  - `update`:  Use a single comment to display cost estimates, creating one if none exist. The GitHub comments UI can be used to see when/what has changed when a comment is updated. PR followers will only be notified on the comment create (not update), and the comment will stay at the same location in the comment history.
  - `delete_and_new`: Delete previous cost estimate comments and create a new one. PR followers will be notified on each comment.
  - `hide_and_new`: Minimize previous cost estimate comments and create a new one. PR followers will be notified on each comment.
  - `new`:  Create a new cost estimate comment. PR followers will be notified on each comment.

- `targetType`: Optional. Which objects should be commented on. May be 'pr' or 'commit'.

- `GITHUB_TOKEN`: Optional, default to `${{ github.token }}`.

## Outputs

This action does not set any direct outputs.
