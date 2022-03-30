# Slack Example

This example shows how to send cost estimates to Slack by combining the Infracost actions with the official [slackapi/slack-github-action](https://github.com/slackapi/slack-github-action) repo.

Slack message blocks have a 3000 char limit so the Infracost CLI automatically truncates the middle of `slack-message` output formats.

For simplicity, this is based off the terraform-plan-json example, which does not require Terraform to be installed.

<img src="/.github/assets/slack-message.png" alt="Example screenshot" />

[//]: <> (BEGIN EXAMPLE)
```yml
name: Slack
on: [pull_request]

jobs:
  slack:
    name: Slack
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v2

      - name: Setup Infracost
        uses: infracost/actions/setup@v1
        with:
          api-key: ${{ secrets.INFRACOST_API_KEY }}

      - name: Generate Infracost JSON
        run: infracost breakdown --path=examples/slack/code/plan.json --format json --out-file /tmp/infracost.json

      - name: Post Infracost comment
        run: |
          # Posts a comment to the PR using the 'update' behavior.
          # This creates a single comment and updates it. The "quietest" option.
          # The other valid behaviors are:
          #   delete-and-new - Delete previous comments and create a new one.
          #   hide-and-new - Minimize previous comments and create a new one.
          #   new - Create a new cost estimate comment on every push.
          # See https://www.infracost.io/docs/features/cli_commands/#comment-on-pull-requests for other options.
          infracost comment github --path /tmp/infracost.json \
                                   --repo $GITHUB_REPOSITORY \
                                   --github-token ${{github.token}} \
                                   --pull-request ${{github.event.pull_request.number}} \
                                   --behavior update

      - name: Generate Slack message
        id: infracost-slack
        run: |
          echo "::set-output name=slack-message::$(infracost output --path /tmp/infracost.json --format slack-message --show-skipped)"
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
