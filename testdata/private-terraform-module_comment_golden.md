
💰 Infracost estimate: **monthly cost will increase by $11.37 📈**
<table>
  <thead>
    <td>Project</td>
    <td>Previous</td>
    <td>New</td>
    <td>Diff</td>
  </thead>
  <tbody>
    <tr>
      <td>infracost/actions/examples/private-terraform-module/code</td>
      <td align="right">$0</td>
      <td align="right">$11.37</td>
      <td>+$11.37</td>
    </tr>
  </tbody>
</table>

<details>
<summary><strong>Infracost output</strong></summary>

```
Project: infracost/actions/examples/private-terraform-module/code

+ module.ec2_cluster.aws_instance.this[0]
  +$11.37

    + Instance usage (Linux/UNIX, on-demand, t2.micro)
      +$8.47

    + EC2 detailed monitoring
      +$2.10

    + root_block_device
    
        + Storage (general purpose SSD, gp2)
          +$0.80

Monthly cost change for infracost/actions/examples/private-terraform-module/code
Amount:  +$11.37 ($0.00 → $11.37)

──────────────────────────────────
Key: ~ changed, + added, - removed

1 cloud resource was detected:
∙ 1 was estimated, 1 includes usage-based costs, see https://infracost.io/usage-file

Add cost estimates to your pull requests: https://infracost.io/cicd
```
</details>
