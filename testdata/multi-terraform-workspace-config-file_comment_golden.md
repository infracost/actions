
💰 Infracost estimate: **monthly cost will increase by $800 📈**
<table>
  <thead>
    <td>Project</td>
    <td>Previous</td>
    <td>New</td>
    <td>Diff</td>
  </thead>
  <tbody>
    <tr>
      <td>infracost/actions/examples/multi-terraform-workspace/code (dev)</td>
      <td align="right">$0</td>
      <td align="right">$51.97</td>
      <td>+$51.97</td>
    </tr>
    <tr>
      <td>infracost/actions/examples/multi-terraform-workspace/code (prod)</td>
      <td align="right">$0</td>
      <td align="right">$748</td>
      <td>+$748</td>
    </tr>
    <tr>
      <td>All projects</td>
      <td align="right">$0</td>
      <td align="right">$800</td>
      <td>+$800</td>
    </tr>
  </tbody>
</table>

<details>
<summary><strong>Infracost output</strong></summary>

```
Project: infracost/actions/examples/multi-terraform-workspace/code (dev)

+ aws_instance.web_app
  +$51.97

    + Instance usage (Linux/UNIX, on-demand, t2.micro)
      +$8.47

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

Monthly cost change for infracost/actions/examples/multi-terraform-workspace/code (dev)
Amount:  +$51.97 ($0.00 → $51.97)

──────────────────────────────────
Project: infracost/actions/examples/multi-terraform-workspace/code (prod)

+ aws_instance.web_app
  +$748

    + Instance usage (Linux/UNIX, on-demand, m5.4xlarge)
      +$561

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

Monthly cost change for infracost/actions/examples/multi-terraform-workspace/code (prod)
Amount:  +$748 ($0.00 → $748)

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
