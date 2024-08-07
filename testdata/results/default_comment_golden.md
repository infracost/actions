
<h4>💰 Infracost report</h4>
<h4>Monthly estimate increased by $586 📈</h4>
<table>
  <thead>
    <td>Changed project</td>
    <td><span title="Baseline costs are consistent charges for provisioned resources, like the hourly cost for a virtual machine, which stays constant no matter how much it is used. Infracost estimates these resources assuming they are used for the whole month (730 hours).">Baseline cost</span></td>
    <td><span title="Usage costs are charges based on actual usage, like the storage cost for an object storage bucket. Infracost estimates these resources using the monthly usage values in the usage-file.">Usage cost</span>*</td>
    <td>Total change</td>
    <td>New monthly cost</td>
  </thead>
  <tbody>
    <tr>
      <td>infracost/actions/testdata/code/example-project/dev</td>
      <td align="right">+$25</td>
      <td align="right">-</td>
      <td align="right">+$25 (+49%)</td>
      <td align="right">$77</td>
    </tr>
    <tr>
      <td>infracost/actions/testdata/code/example-project/prod</td>
      <td align="right">+$561</td>
      <td align="right">-</td>
      <td align="right">+$561 (+75%)</td>
      <td align="right">$1,308</td>
    </tr>
  </tbody>
</table>


*Usage costs can be estimated by updating [Infracost Cloud settings](https://www.infracost.io/docs/features/usage_based_resources), see [docs](https://www.infracost.io/docs/features/usage_based_resources/#infracost-usageyml) for other options.
<details>

<summary>Estimate details </summary>

```
Key: * usage cost, ~ changed, + added, - removed

──────────────────────────────────
Project: dev
Module path: dev

~ module.base.aws_instance.web_app
  +$25 ($52 → $77)

    ~ Instance usage (Linux/UNIX, on-demand, t2.micro → t2.medium)
      +$25 ($8 → $34)

Monthly cost change for infracost/actions/testdata/code/example-project/dev (Module path: dev)
Amount:  +$25 ($52 → $77)
Percent: +49%

──────────────────────────────────
Project: prod
Module path: prod

~ module.base.aws_instance.web_app
  +$561 ($748 → $1,308)

    ~ Instance usage (Linux/UNIX, on-demand, m5.4xlarge → m5.8xlarge)
      +$561 ($561 → $1,121)

Monthly cost change for infracost/actions/testdata/code/example-project/prod (Module path: prod)
Amount:  +$561 ($748 → $1,308)
Percent: +75%

──────────────────────────────────
Key: * usage cost, ~ changed, + added, - removed

*Usage costs can be estimated by updating Infracost Cloud settings, see docs for other options.

4 cloud resources were detected:
∙ 4 were estimated

Infracost estimate: Monthly estimate increased by $586 ↑
┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━┳━━━━━━━━━━━━━━┓
┃ Changed project                                      ┃ Baseline cost ┃ Usage cost* ┃ Total change ┃
┣━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━╋━━━━━━━━━━━━━━━╋━━━━━━━━━━━━━╋━━━━━━━━━━━━━━┫
┃ infracost/actions/testdata/code/example-project/dev  ┃          +$25 ┃           - ┃  +$25 (+49%) ┃
┃ infracost/actions/testdata/code/example-project/prod ┃         +$561 ┃           - ┃ +$561 (+75%) ┃
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┻━━━━━━━━━━━━━━━┻━━━━━━━━━━━━━┻━━━━━━━━━━━━━━┛
```
</details>
<sub>This comment will be updated when code changes.
</sub>

Comment not posted to GitHub (--dry-run was specified)
