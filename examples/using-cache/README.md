# Terraform/Terragrunt project (using the GitHub Actions cache)

This example is similar to the [Terraform/Terragrunt project (single or multi)](../terraform-project) example but uses the [GitHub Actions cache](https://github.com/actions/cache) to cache the baseline result and any downloaded remote modules. This should speed up the workflow for projects that use a lot of external modules. The first time the pipeline runs it downloads all remote modules, but for subsequent runs it will retrieve them from the cache.

If you have any issues running this workflow please reach out to us on [our community Slack channel](https://www.infracost.io/community-chat).

[//]: <> (BEGIN EXAMPLE)
```yml
name: Terraform project (using the GitHub Actions cache)
on:
  # Run on all pull requests and pushes to the main/master branch so we can cache the baseline results.
  pull_request:
  push:
    branches:
      - main
      - master
env:
  # If you use private modules you'll need this env variable to use
  # the same ssh-agent socket value across all jobs & steps.
  SSH_AUTH_SOCK: /tmp/ssh_agent.sock
jobs:
  terraform-project-using-cache:
    name: Terraform project (using the GitHub Actions cache)
    runs-on: ubuntu-latest
    permissions:
      contents: read
      pull-requests: write

    env:
      TF_ROOT: examples/terraform-project/code
      # This instructs the CLI to send cost estimates to Infracost Cloud. Our SaaS product
      #   complements the open source CLI by giving teams advanced visibility and controls.
      #   The cost estimates are transmitted in JSON format and do not contain any cloud
      #   credentials or secrets (see https://infracost.io/docs/faq/ for more information).
      INFRACOST_ENABLE_CLOUD: true
      # If you're using Terraform Cloud/Enterprise and have variables or private modules stored
      # on there, specify the following to automatically retrieve the variables:
      #   INFRACOST_TERRAFORM_CLOUD_TOKEN: ${{ secrets.TFC_TOKEN }}
      #   INFRACOST_TERRAFORM_CLOUD_HOST: app.terraform.io # Change this if you're using Terraform Enterprise

    steps:
      # If you use private modules, add an environment variable or secret
      # called GIT_SSH_KEY with your private key, so Infracost can access
      # private repositories (similar to how Terraform/Terragrunt does).
      # - name: add GIT_SSH_KEY
      #   run: |
      #     ssh-agent -a $SSH_AUTH_SOCK
      #     mkdir -p ~/.ssh
      #     echo "${{ secrets.GIT_SSH_KEY }}" | tr -d '\r' | ssh-add -
      #     ssh-keyscan github.com >> ~/.ssh/known_hosts

      - name: Setup Infracost
        uses: infracost/actions/setup@v2
        # See https://github.com/infracost/actions/tree/master/setup for other inputs
        # If you can't use this action, see Docker images in https://infracost.io/cicd
        with:
          api-key: ${{ secrets.INFRACOST_API_KEY }}

      - name: Cache the Infracost baseline JSON result
        id: cache-infracost-base-json
        uses: actions/cache@v3
        with:
          path: '/tmp/infracost-base.json'
          key: infracost-base-json-${{ runner.os }}-${{ github.event.pull_request.base.sha || github.sha }}

      # Checkout the base branch of the pull request (e.g. main/master).
      - name: Checkout base branch
        uses: actions/checkout@v2
        with:
          ref: '${{ github.event.pull_request.base.ref }}'

      # Downloading remote Terraform modules can be slow, so we add them to the GitHub cache.
      # We skip this for pushes to the main/master branch that already have the baseline generated.
      - name: Cache .infracost/terraform_modules for target branch
        uses: actions/cache@v3
        with:
          path: |
            ${{ env.TF_ROOT }}/**/.infracost/terraform_modules/**
            !${{ env.TF_ROOT }}/**/.infracost/terraform_modules/**/.git/**
          key: infracost-terraform-modules-${{ runner.os }}-${{ github.event.pull_request.base.sha || github.sha }}
          # If there's no cached record for this commit, pull in the latest cached record anyway
          # Internally infracost will downloaded any additional required modules if required
          restore-keys: infracost-terraform-modules-${{ runner.os }}-
        if: github.event_name == 'pull_request' || steps.cache-infracost-base-json.outputs.cache-hit != 'true'

      # Generate Infracost JSON file as the baseline. We skip this if we've already generated one for this SHA.
      # This will also run on pull request pushes if we get a cache miss to catch cases where
      # the baseline run hasn't been run on main/master yet.
      - name: Generate Infracost cost estimate baseline
        run: |
          infracost breakdown --path=${TF_ROOT} \
                              --format=json \
                              --out-file=/tmp/infracost-base.json
        if: steps.cache-infracost-base-json.outputs.cache-hit != 'true'

      # Checkout the current PR branch so we can create a diff.
      - name: Checkout PR branch
        uses: actions/checkout@v2
        with:
          # Make sure the .infracost dir stays between runs so that downloaded modules persist
          clean: false
        if: github.event_name == 'pull_request'

      # Generate an Infracost diff and save it to a JSON file.
      - name: Generate Infracost diff
        run: |
          infracost diff --path=${TF_ROOT} \
                          --format=json \
                          --compare-to=/tmp/infracost-base.json \
                          --out-file=/tmp/infracost.json
        if: github.event_name == 'pull_request'

      # Posts a comment to the PR using the 'update' behavior.
      # This creates a single comment and updates it. The "quietest" option.
      # The other valid behaviors are:
      #   delete-and-new - Delete previous comments and create a new one.
      #   hide-and-new - Minimize previous comments and create a new one.
      #   new - Create a new cost estimate comment on every push.
      # See https://www.infracost.io/docs/features/cli_commands/#comment-on-pull-requests for other options.
      - name: Post Infracost comment
        run: |
            infracost comment github --path=/tmp/infracost.json \
                                     --repo=$GITHUB_REPOSITORY \
                                     --github-token=${{github.token}} \
                                     --pull-request=${{github.event.pull_request.number}} \
                                     --behavior=update
        if: github.event_name == 'pull_request'
```
[//]: <> (END EXAMPLE)
