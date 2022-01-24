# Open Policy Agent Example

This example shows how to set cost policies using [Open Policy Agent](https://www.openpolicyagent.org/).  For simplicity, this is based off the terraform-plan-json example, which does not require Terraform to be installed.

When the policy checks pass, the GitHub Action step called "Check Policies" passes and outputs `Policy check passed.` in the action logs. When the policy checks fail, that step fails and the action logs show the details of the failing policies.

Create a policy file (e.g. `policy.rego`) that checks the Infracost JSON: 
```rego
package infracost

# totalDiff
deny[msg] {
	maxDiff = 1500.0
	to_number(input.diffTotalMonthlyCost) >= maxDiff

	msg := sprintf(
		"Total monthly cost diff must be less than $%.2f (actual diff is $%.2f)",
		[maxDiff, to_number(input.diffTotalMonthlyCost)],
	)
}

# instanceCost
deny[msg] {
	r := input.projects[_].breakdown.resources[_]
	startswith(r.name, "aws_instance.")

	maxHourlyCost := 2.0
	to_number(r.hourlyCost) > maxHourlyCost

	msg := sprintf(
		"AWS instances must cost less than $%.2f\\hr (%s costs $%.2f\\hr).",
		[maxHourlyCost, r.name, to_number(r.hourlyCost)],
	)
}

# instanceIOPSCost
deny[msg] {
	r := input.projects[_].breakdown.resources[_]
	startswith(r.name, "aws_instance.")

	baseHourlyCost := to_number(r.costComponents[_].hourlyCost)

	sr_cc := r.subresources[_].costComponents[_]
	sr_cc.name == "Provisioned IOPS"
	iopsHourlyCost := to_number(sr_cc.hourlyCost)

	iopsHourlyCost > baseHourlyCost

	msg := sprintf(
		"AWS instance IOPS must cost less than compute usage (%s IOPS $%.2f\\hr, usage $%.2f\\hr).",
		[r.name, iopsHourlyCost, baseHourlyCost],
	)
}
```

Then use OPA to test infrastructure cost changes against the policy.

[//]: <> (BEGIN EXAMPLE)
```yml
name: Open Policy Agent
on: [pull_request]

jobs:
  open-policy-agent:
    name: Open Policy Agent
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v2
      
      - name: Setup Infracost
        uses: infracost/actions/setup@v1
        with:
          api-key: ${{ secrets.INFRACOST_API_KEY }}

      - name: Setup OPA
        uses: open-policy-agent/setup-opa@v1

      - name: Run Infracost
        run: infracost breakdown --path=examples/opa/code/plan.json --format=json --out-file=/tmp/infracost.json
        
      - name: Run OPA
        run: opa eval --input /tmp/infracost.json -d examples/opa/policy/policy.rego --format pretty "data.infracost.deny" | tee /tmp/opa.out

      - name: Check Policies
        run: |
          denyReasons=$(</tmp/opa.out)
          if [ "$denyReasons" != "[]" ]; then
            echo -e "::error::Policy check failed:\n$denyReasons"
            exit 1
          else
            echo "::info::Policy check passed."
          fi
```
[//]: <> (END EXAMPLE)
