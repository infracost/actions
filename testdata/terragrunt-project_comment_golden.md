
💰 Infracost estimate: **monthly cost will increase by $1,386 📈**
<table>
  <thead>
    <td>Project</td>
    <td>Previous</td>
    <td>New</td>
    <td>Diff</td>
  </thead>
  <tbody>
    <tr>
      <td>infracost/actions/examples/plan.../terragrunt/code/dev/plan.json</td>
      <td align="right">$0</td>
      <td align="right">$77.37</td>
      <td>+$77.37</td>
    </tr>
    <tr>
      <td>infracost/actions/examples/plan...terragrunt/code/prod/plan.json</td>
      <td align="right">$0</td>
      <td align="right">$1,308</td>
      <td>+$1,308</td>
    </tr>
    <tr>
      <td>All projects</td>
      <td align="right">$0</td>
      <td align="right">$1,386</td>
      <td>+$1,386</td>
    </tr>
  </tbody>
</table>

<details>
<summary><strong>Infracost output</strong></summary>

```
Project: infracost/actions/examples/plan-json/terragrunt/code/dev/plan.json

+ aws_instance.web_app
  +$77.37

    + Instance usage (Linux/UNIX, on-demand, t2.medium)
      +$33.87

    + root_block_device
    
        + Storage (general purpose SSD, gp2)
          +$5.00

    + ebs_block_device[0]
    
        + Storage (provisioned IOPS SSD, io1)
          +$12.50
    
        + Provisioned IOPS
          +$26.00

+ aws_lambda_function.hello_world
  Monthly cost depends on usage

    + Requests
      Monthly cost depends on usage
        +$0.20 per 1M requests

    + Duration
      Monthly cost depends on usage
        +$0.0000166667 per GB-seconds

Monthly cost change for infracost/actions/examples/plan-json/terragrunt/code/dev/plan.json
Amount:  +$77.37 ($0.00 → $77.37)

──────────────────────────────────
Project: infracost/actions/examples/plan-json/terragrunt/code/prod/plan.json

+ aws_instance.web_app
  +$1,308

    + Instance usage (Linux/UNIX, on-demand, m5.8xlarge)
      +$1,121

    + root_block_device
    
        + Storage (general purpose SSD, gp2)
          +$10.00

    + ebs_block_device[0]
    
        + Storage (provisioned IOPS SSD, io1)
          +$125
    
        + Provisioned IOPS
          +$52.00

+ aws_lambda_function.hello_world
  Monthly cost depends on usage

    + Requests
      Monthly cost depends on usage
        +$0.20 per 1M requests

    + Duration
      Monthly cost depends on usage
        +$0.0000166667 per GB-seconds

Monthly cost change for infracost/actions/examples/plan-json/terragrunt/code/prod/plan.json
Amount:  +$1,308 ($0.00 → $1,308)

──────────────────────────────────
Key: ~ changed, + added, - removed

4 cloud resources were detected:
∙ 4 were estimated, all of which include usage-based costs, see https://infracost.io/usage-file
```
</details>

This comment will be updated when the cost estimate changes.

<sub>
  Is this comment useful? <a href="https://www.infracost.io/feedback/submit/?value=yes" rel="noopener noreferrer" target="_blank">Yes</a>, <a href="https://www.infracost.io/feedback/submit/?value=no" rel="noopener noreferrer" target="_blank">No</a>
</sub>

Comment not posted to GitHub (--dry-run was specified)
