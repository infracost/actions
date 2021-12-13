# Conftest Example

This example shows how to set thresholds and cost policies using conftest.  For simplicity, this is based off the terraform-plan-json example, which does not require Terraform to be installed.

Create a policy file that checks the infracost JSON: 
```rego
package main

deny_totalDiff[msg] {
  to_number(input.diffTotalMonthlyCost) >= 1500

  msg := sprintf("Total monthly cost diff must be < $1500 (actual diff is $%.2f)", [to_number(input.diffTotalMonthlyCost)])
}

deny_instanceCost[msg] {
	r := input.projects[_].breakdown.resources[_]
  startswith(r.name, "aws_instance.")

  maxHourlyCost := 2.0
  to_number(r.hourlyCost) > maxHourlyCost

  msg :=  sprintf("AWS instances must cost less than $%.2f\\hr (%s costs $%.2f\\hr).", [maxHourlyCost, r.name, to_number(r.hourlyCost)])
}

deny_instanceCost[msg] {
	r := input.projects[_].breakdown.resources[_]
  startswith(r.name, "aws_instance.")

  baseHourlyCost := to_number(r.costComponents[_].hourlyCost)

  sr_cc := r.subresources[_].costComponents[_]
  sr_cc.name == "Provisioned IOPS"
  iopsHourlyCost := to_number(sr_cc.hourlyCost)

  iopsHourlyCost > baseHourlyCost

  msg :=  sprintf("AWS instance IOPS must cost less than the instance usage (%s IOPS costs $%.2f\\hr, usage costs $%.2f\\hr).", [r.name, iopsHourlyCost, baseHourlyCost])
}
```

Then use conftest to test check infrastructure cost changes against the policy.

[//]: <> (BEGIN EXAMPLE)
```yml
name: Conftest
on: [pull_request]

jobs:
  conftest:
    name: Conftest
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v2
      
      - name: Setup Infracost
        uses: infracost/actions/setup@v1
        with:
          api-key: ${{ secrets.INFRACOST_API_KEY }}

      - name: Setup Conftest
        uses: artis3n/setup-conftest@v0
        with:
          conftest_wrapper: false

      - name: Run Infracost
        run: infracost breakdown --path=examples/conftest/code/plan.json --format=json --out-file=/tmp/infracost.json

      - name: Check Conftest Policies
        id: conftest
        run: conftest test --policy examples/conftest/policy /tmp/infracost.json             
```
[//]: <> (END EXAMPLE)
