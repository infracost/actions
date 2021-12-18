# Sentinel Example

This example shows how to write cost policies with Hashicorp's [Sentinel](https://www.hashicorp.com/sentinel).  For simplicity, this example evaluates policies with the Sentinel CLI (a.k.a. Sentinel Simulator).

When the policy checks pass, the GitHub Action step called "Check Policies" passes and outputs `Policy check passed.` in the action logs. When the policy checks fail, that step fails and the action logs show the Sentinel output indicating failing policies.

Create a policy file (e.g. `policy.policy) that checks a global parameter 'breakdown' containing the Infracost JSON: 
```policy
import "strings"

limitTotalDiff = rule {
    float(breakdown.totalMonthlyCost) < 1500
}

awsInstances = filter breakdown.projects[0].breakdown.resources as _, resource {
	strings.split(resource.name, ".")[0] is "aws_instance"
}

limitInstanceCost = rule {
  all awsInstances as _, instance {
  	float(instance.hourlyCost) <= 2.00
  }
}

instanceBaseCost = func(instance) {
  cost = 0.0
 	for instance.costComponents as cc {
    cost += float(cc.hourlyCost)
  }
  return cost
}

instanceIOPSCost = func(instance) {
  cost = 0.0
 	for instance.subresources as sr {
    for sr.costComponents as cc {
      if cc.name == "Provisioned IOPS" {
        cost += float(cc.hourlyCost)
      }
    }
  }
  return cost
}

limitInstanceIOPSCost = rule {
   all awsInstances as _, instance {
  	instanceIOPSCost(instance) <= instanceBaseCost(instance)
  }
}

main = rule {
    limitTotalDiff and
    limitInstanceCost and
    limitInstanceIOPSCost
}
```

Then use Sentinel to test infrastructure cost changes against the policy.

[//]: <> (BEGIN EXAMPLE)
```yml
name: Sentinel
on: [pull_request]

jobs:
  sentinel:
    name: Sentinel
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v2
      
      - name: Setup Infracost
        uses: infracost/actions/setup@v1
        with:
          api-key: ${{ secrets.INFRACOST_API_KEY }}

      - name: Setup Sentinel
        uses: innovationnorway/setup-sentinel@v1

      - name: Run Infracost
        run: infracost breakdown --path=examples/sentinel/code/plan.json --format=json --out-file=/tmp/infracost.json
        
      - name: Run Sentinel
        run: sentinel apply -global breakdown="$(cat /tmp/infracost.json)" examples/sentinel/policy/policy.policy | tee /tmp/sentinel.out

      - name: Check Policies
        run: |
          result=$(</tmp/sentinel.out)
          if [ "$result" != "Pass - policy.policy" ]; then
            echo -e "::error::Policy check failed:\n$result"
            exit 1
          else
            echo "::info::Policy check passed."
          fi
```
[//]: <> (END EXAMPLE)
