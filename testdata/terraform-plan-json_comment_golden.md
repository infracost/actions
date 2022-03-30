
💰 Infracost estimate: **monthly cost will increase by $743 📈**
<table>
  <thead>
    <td>Project</td>
    <td>Previous</td>
    <td>New</td>
    <td>Diff</td>
  </thead>
  <tbody>
    <tr>
      <td>infracost/actions/examples/terraform-plan-json/code/plan.json</td>
      <td align="right">$0</td>
      <td align="right">$743</td>
      <td>+$743</td>
    </tr>
  </tbody>
</table>

<details>
<summary><strong>Infracost output</strong></summary>

```
Project: infracost/actions/examples/terraform-plan-json/code/plan.json

+ aws_instance.web_app
  +$743

    + Instance usage (Linux/UNIX, on-demand, m5.4xlarge)
      +$561

    + root_block_device
    
        + Storage (general purpose SSD, gp2)
          +$5.00

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

Monthly cost change for infracost/actions/examples/terraform-plan-json/code/plan.json
Amount:  +$743 ($0.00 → $743)

──────────────────────────────────
Key: ~ changed, + added, - removed

2 cloud resources were detected:
∙ 2 were estimated, all of which include usage-based costs, see https://infracost.io/usage-file
```
</details>

This comment will be updated when the cost estimate changes.

<sub>
  Is this comment useful? <a href="https://www.infracost.io/feedback/submit/?value=yes" rel="noopener noreferrer" target="_blank">Yes</a>, <a href="https://www.infracost.io/feedback/submit/?value=no" rel="noopener noreferrer" target="_blank">No</a>
</sub>

Comment not posted to GitHub (--dry-run was specified)
