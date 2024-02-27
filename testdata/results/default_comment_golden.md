
<h3>Infracost report</h3>
<h4>💰 Monthly cost will increase by $586 📈</h4>
<table>
  <thead>
    <td>Project</td>
    <td>Cost change</td>
    <td>New monthly cost</td>
  </thead>
  <tbody>
    <tr>
      <td>infracost/actions/testdata/code/example-project/dev</td>
      <td>+$25 (+49%)</td>
      <td align="right">$77</td>
    </tr>
    <tr>
      <td>infracost/actions/testdata/code/example-project/prod</td>
      <td>+$561 (+75%)</td>
      <td align="right">$1,308</td>
    </tr>
  </tbody>
</table>
<details>
<summary>Cost details</summary>

```
──────────────────────────────────
Project: infracost/actions/testdata/code/example-project/dev
Module path: dev

~ module.base.aws_instance.web_app
  +$25 ($52 → $77)

    ~ Instance usage (Linux/UNIX, on-demand, t2.micro → t2.medium)
      +$25 ($8 → $34)

Monthly cost change for infracost/actions/testdata/code/example-project/dev (Module path: dev)
Amount:  +$25 ($52 → $77)
Percent: +49%

──────────────────────────────────
Project: infracost/actions/testdata/code/example-project/prod
Module path: prod

~ module.base.aws_instance.web_app
  +$561 ($748 → $1,308)

    ~ Instance usage (Linux/UNIX, on-demand, m5.4xlarge → m5.8xlarge)
      +$561 ($561 → $1,121)

Monthly cost change for infracost/actions/testdata/code/example-project/prod (Module path: prod)
Amount:  +$561 ($748 → $1,308)
Percent: +75%

──────────────────────────────────
Key: ~ changed, + added, - removed

4 cloud resources were detected:
∙ 4 were estimated, all of which include usage-based costs, see https://infracost.io/usage-file

Infracost estimate: Monthly cost will increase by $586 ↑
┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━━┓
┃ Project                                              ┃ Cost change  ┃ New monthly cost ┃
┣━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━╋━━━━━━━━━━━━━━╋━━━━━━━━━━━━━━━━━━┫
┃ infracost/actions/testdata/code/example-project/dev  ┃  +$25 (+49%) ┃ $77              ┃
┃ infracost/actions/testdata/code/example-project/prod ┃ +$561 (+75%) ┃ $1,308           ┃
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┻━━━━━━━━━━━━━━┻━━━━━━━━━━━━━━━━━━┛
```
</details>
<sub>This comment will be updated when code changes.
</sub>

Comment not posted to GitHub (--dry-run was specified)
