# Infracost GitHub Actions

### Try the GitHub App
👉👉 We recommend using the [**free Infracost GitHub App**](https://www.infracost.io/docs/integrations/github_app/) as it's much simpler to setup and faster to run

---

This GitHub Action runs Infracost so you can see cloud cost estimates and FinOps best practices for Terraform in pull requests 💰

<img src=".github/assets/pull-request.png" alt="Example screenshot" />

## Quick start

1. If you haven't done so already, [download Infracost](https://www.infracost.io/docs/#quick-start) and run `infracost auth login` to get a free API key.

2. Retrieve your Infracost API key by running `infracost configure get api_key`.

3. [Create a repo secret](https://docs.github.com/en/actions/configuring-and-managing-workflows/creating-and-storing-encrypted-secrets#creating-encrypted-secrets-for-a-repository) called `INFRACOST_API_KEY` with your API key.

4. Create a new file in `.github/workflows/infracost.yml` in your repo with the following content.

```yaml
# Infracost runs on pull requests (PR) and posts PR comments.
# If you use Infracost Cloud, Infracost also runs on main/master branch pushes so the dashboard is updated.
# The GitHub Action docs (https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows) describe other trigger options.
on:
  pull_request:
    types: [opened, synchronize, closed]
  push:
    branches:
      - main
      - master

env:
  # If you use private modules you'll need this env variable to use
  # the same ssh-agent socket value across all jobs & steps.
  SSH_AUTH_SOCK: /tmp/ssh_agent.sock
jobs:
  # This stage runs the Infracost CLI and posts PR comments.
  # It also updates PR comments when the PR is updated (synchronize event).
  infracost-pull-request-checks:
    name: Infracost Pull Request Checks
    if: github.event_name == 'pull_request' && (github.event.action == 'opened' || github.event.action == 'synchronize')
    runs-on: ubuntu-latest
    permissions:
      contents: read
      pull-requests: write # Required to post comments
    # env:
      # If you store Terraform variables or modules in a 3rd party such as TFC or Spacelift,
      # specify the following so Infracost can automatically retrieve them.
      # See https://www.infracost.io/docs/features/terraform_modules/#registry-modules for details.
      #   INFRACOST_TERRAFORM_CLOUD_TOKEN: ${{ secrets.TFC_TOKEN }}
      #   INFRACOST_TERRAFORM_CLOUD_HOST: app.terraform.io
    steps:
      # If you use private modules, add an environment variable or secret
      # called GIT_SSH_KEY with your private key, so Infracost CLI can access
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
        # If you can't use this action, use Docker image infracost/infracost:ci-0.10
        with:
          api-key: ${{ secrets.INFRACOST_API_KEY }}

      # Checkout the base branch of the pull request (e.g. main/master).
      - name: Checkout base branch
        uses: actions/checkout@v4
        with:
          ref: '${{ github.event.pull_request.base.ref }}'

      # Generate Infracost JSON file as the baseline.
      - name: Generate Infracost cost estimate baseline
        run: |
          infracost breakdown --path=. \
                              --format=json \
                              --out-file=/tmp/infracost-base.json

      # Checkout the current PR branch so we can create a diff.
      - name: Checkout PR branch
        uses: actions/checkout@v4

      # Generate an Infracost diff and save it to a JSON file.
      - name: Generate Infracost diff
        run: |
          infracost diff --path=. \
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
      - name: Post Infracost comment
        run: |
            infracost comment github --path=/tmp/infracost.json \
                                     --repo=$GITHUB_REPOSITORY \
                                     --github-token=${{ github.token }} \
                                     --pull-request=${{ github.event.pull_request.number }} \
                                     --behavior=update

  # Run Infracost on default branch and update Infracost Cloud
  infracost-default-branch-update:
    # If you use private modules, or store Terraform variables or modules in a 3rd party
    # such as TFC or Spacelift, include the same steps/variables as the infracost-pull-request-checks job
    name: Infracost Default Branch Update
    if: github.event_name == 'push' && (github.ref_name == 'main' || github.ref_name == 'master')
    runs-on: ubuntu-latest
    steps:
      - name: Setup Infracost
        uses: infracost/actions/setup@v2
        with:
          api-key: ${{ secrets.INFRACOST_API_KEY }}

      - name: Checkout main/master branch
        uses: actions/checkout@v4

      - name: Run Infracost on default branch and update Infracost Cloud
        run: |
          infracost breakdown --path=. \
                    --format=json \
                    --out-file=infracost.json

          infracost upload --path=infracost.json || echo "Always pass main branch runs even if there are policy failures"

  # Update PR status in Infracost Cloud
  infracost-pull-request-status-update:
    name: Infracost PR Status Update
    if: github.event_name == 'pull_request' && github.event.action == 'closed'
    runs-on: ubuntu-latest
    steps:
    - name: Infracost PR Status Update
      run: |
        PR_STATUS="MERGED"
        if [[ ${{ github.event.pull_request.merged }} = false ]]; then PR_STATUS="CLOSED"; fi

        echo "Updating status of ${{ github.event.pull_request.html_url }} to $PR_STATUS"
        curl -i \
          --request POST \
          --header "Content-Type: application/json" \
          --header "X-API-Key: $INFRACOST_API_KEY" \
          --data "{ \"query\": \"mutation {updatePullRequestStatus( url: \\\"${{ github.event.pull_request.html_url }}\\\", status: $PR_STATUS )}\" }" \
          "https://dashboard.api.infracost.io/graphql";
      env:
        INFRACOST_API_KEY: ${{ secrets.INFRACOST_API_KEY }}
```

5. Follow [these simple steps](https://www.infracost.io/docs/infracost_cloud/get_started/#4-send-a-pull-request) to test the integration. This is important as it ensures the CLI commands are running correctly in your workflow 👌

6. [Infracost Cloud](https://dashboard.infracost.io) is our SaaS product that builds on top of Infracost open source. It enables team leads, managers and FinOps practitioners to setup [tagging policies](https://www.infracost.io/docs/infracost_cloud/tagging_policies/), [guardrails](https://www.infracost.io/docs/infracost_cloud/guardrails/) and [best practices](https://www.infracost.io/docs/infracost_cloud/cost_policies/) to help guide the team. For example, you can check for required tag keys/values, or suggest switching AWS gp2 volumes to gp3 as they are more performant and cheaper.

    If you **do not** want to use [Infracost Cloud](https://dashboard.infracost.io), go to Org Settings and disable the dashboard. This causes the CLI not to send its JSON output to your dashboard; the JSON does not contain any cloud credentials or secrets, see the [FAQ](https://infracost.io/docs/faq/) for more information.

    <img src=".github/assets/infracost-cloud-dashboard.png" alt="Infracost Cloud enables you to check for best practices such as using latest generation instance types or block storage, as well as setup tagging policies and guardrails to help guide the team." />

## Troubleshooting

If you run into any issues, check the GitHub Action logs and the following steps. You can also join our [community Slack channel](https://www.infracost.io/community-chat), we'll help you very quickly 😄

### Permissions issue

If you receive an error when running the `infracost comment` command in your pipeline, it's probably related to `${{ github.token }}`. This is the default GitHub token available to actions and is used to post comments. The default [token permissions](https://docs.github.com/en/actions/learn-github-actions/workflow-syntax-for-github-actions#permissions) are read-only by default and `pull-requests: write` is required. If you are using SAML single sign-on, you must first [authorize the token](https://docs.github.com/en/enterprise-cloud@latest/authentication/authenticating-with-saml-single-sign-on/authorizing-a-personal-access-token-for-use-with-saml-single-sign-on).

#### The `add GIT_SSH_KEY` step fails

If you are using private modules and receive a `option requires an argument -- a` error in the `add GIT_SSH_KEY` step:
1. Make sure you have the following set in your workflow `SSH_AUTH_SOCK`:
    ```yml
    env:
      SSH_AUTH_SOCK: /tmp/ssh_agent.sock
    ```
2. Try changing the `ssh-agent -a $SSH_AUTH_SOCK` line to the following:
    ```yml
    ssh-agent -a "${{ env.SSH_AUTH_SOCK }}"
    ```

## Contributing

Issues and pull requests are welcome! For major changes, including interface changes, please open an issue first to discuss what you would like to change. [Join our community Slack channel](https://www.infracost.io/community-chat), we are a friendly bunch and happy to help you get started :)

## License

[Apache License 2.0](https://choosealicense.com/licenses/apache-2.0/)
