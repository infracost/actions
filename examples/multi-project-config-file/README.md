# Multi-project using config file

This example shows how to run Infracost in GitHub Actions with multiple Terraform projects using a config file. The [config file]((https://www.infracost.io/docs/config_file/)) can be used to specify variable files or the Terraform workspaces for different projects.

[//]: <> (BEGIN EXAMPLE)
```yml
name: Multi-project config file
on: [pull_request]
env:
  # If you use private modules you'll need this env variable to use 
  # the same ssh-agent socket value across all jobs & steps. 
  SSH_AUTH_SOCK: /tmp/ssh_agent.sock

jobs:
  multi-project-config-file:
    name: Multi-project config file
    runs-on: ubuntu-latest
    env:
      TF_ROOT: examples/multi-project-config-file/code

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

      # Checkout the branch you want Infracost to compare costs against.This example is using the
      # target PR branch.
      - name: Checkout base branch
        uses: actions/checkout@v2
        with:
          ref: '${{ github.event.pull_request.base.ref }}'

      # Generate Infracost JSON file as the baseline.
      - name: Generate Infracost cost estimate baseline
        run: |
          infracost breakdown --config-file=${TF_ROOT}/infracost.yml \
                              --format=json \
                              --out-file=/tmp/infracost-base.json

      # Checkout the current PR branch so we can create a diff.
      - name: Checkout PR branch
        uses: actions/checkout@v2

      # Generate an Infracost diff and save it to a JSON file.
      - name: Generate Infracost diff
        run: |
          infracost diff --config-file=${TF_ROOT}/infracost.yml \
                              --format=json \
                              --compare-to=/tmp/infracost-base.json \
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
          INFRACOST_ENABLE_CLOUD=true infracost comment github --path=/tmp/infracost.json \
                                   --repo=$GITHUB_REPOSITORY \
                                   --github-token=${{github.token}} \
                                   --pull-request=${{github.event.pull_request.number}} \
                                   --behavior=update
```
[//]: <> (END EXAMPLE)
