# Slack Example

This example shows how to send cost estimates to Slack by combining the Infracost GitHub Action with the official [slackapi/slack-github-action](https://github.com/slackapi/slack-github-action) repo.

Slack message blocks have a 3000 char limit so the Infracost CLI automatically truncates the middle of `slack-message` output formats.

<img src="/.github/assets/slack-message.png" alt="Example screenshot" />

[//]: <> (BEGIN EXAMPLE)
```yml
name: Slack
on: [pull_request]

jobs:
  slack:
    name: Slack
    runs-on: ubuntu-latest
    env:
      TF_ROOT: examples/terraform-project/code

    steps:
      - name: Setup Infracost
        uses: infracost/actions/setup@v2
        with:
          api-key: ${{ secrets.INFRACOST_API_KEY }}

      # Checkout the base branch of the pull request (e.g. main/master).
      - name: Checkout base branch
        uses: actions/checkout@v2
        with:
          ref: '${{ github.event.pull_request.base.ref }}'

      # Generate Infracost JSON file as the baseline.
      - name: Generate Infracost cost estimate baseline
        run: |
          infracost breakdown --path=${TF_ROOT} \
                              --format=json \
                              --out-file=/tmp/infracost-base.json

      # Checkout the current PR branch so we can create a diff.
      - name: Checkout PR branch
        uses: actions/checkout@v2

      # Generate an Infracost diff and save it to a JSON file.
      - name: Generate Infracost diff
        run: |
          infracost diff --path=${TF_ROOT} \
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
                                   --github-token=${{github.token}} \
                                   --pull-request=${{github.event.pull_request.number}} \
                                   --behavior=update

      - name: Generate Slack message
        id: infracost-slack
        run: |
          echo "::set-output name=slack-message::$(infracost output --path=/tmp/infracost.json --format=slack-message --show-skipped)"
          echo "::set-output name=diffTotalMonthlyCost::$(jq '(.diffTotalMonthlyCost // 0) | tonumber' /tmp/infracost.json)"

      - name: Send cost estimate to Slack
        uses: slackapi/slack-github-action@v1
        if: ${{ steps.infracost-slack.outputs.diffTotalMonthlyCost > 0 }} # Only post to Slack if there is a cost diff
        with:
          payload: ${{ steps.infracost-slack.outputs.slack-message }}
        env:
          SLACK_WEBHOOK_URL: ${{ secrets.SLACK_WEBHOOK_URL }}
          SLACK_WEBHOOK_TYPE: INCOMING_WEBHOOK
```
[//]: <> (END EXAMPLE)
