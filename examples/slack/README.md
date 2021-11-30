# Slack Example

This example shows how to send cost estimates to Slack by combining the Infracost actions with the official [slackapi/slack-github-action](https://github.com/slackapi/slack-github-action) repo. 

Slack message blocks have a 3000 char limit so the Infracost CLI automatically truncates the middle of `slack-message` output formats.

For simplicity, this is based off the terraform-plan-json example, which does not require Terraform to be installed.

<img src=".github/assets/slack-message.png" alt="Example screenshot" />

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
        id: infracost-commands
        run: |
          infracost breakdown --path=examples/thresholds/code/plan.json --format json --out-file /tmp/infracost.json
          echo "::set-output name=slack-message::$(infracost output --path /tmp/infracost.json --format slack-message --show-skipped)"
          echo "::set-output name=diffTotalMonthlyCost::$(jq '(.diffTotalMonthlyCost // 0) | tonumber' /tmp/infracost.json)"

      - name: Post the comment
        uses: infracost/actions/comment@v1
        with:
          path: /tmp/infracost.json
          behavior: update # Create a single comment and update it. See https://github.com/infracost/actions/tree/master/comment for other options

      - name: Send cost estimate to Slack
        uses: slackapi/slack-github-action@v1
        if: ${{ steps.infracost-commands.outputs.diffTotalMonthlyCost > 0 }} # Only post to Slack if there is a cost diff
        with:
          payload: ${{ steps.infracost-commands.outputs.slack-message }}
        env:
          SLACK_WEBHOOK_URL: ${{ secrets.SLACK_WEBHOOK_URL }}
          SLACK_WEBHOOK_TYPE: INCOMING_WEBHOOK          
```
[//]: <> (END EXAMPLE)
