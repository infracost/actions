# Sentinel Example

This example shows how to write [cost policies](https://www.infracost.io/docs/features/cost_policies/) with HashiCorp's [Sentinel](https://www.hashicorp.com/sentinel). For simplicity, this example evaluates policies with the Sentinel CLI (a.k.a. Sentinel Simulator). The point of this example is to show how a policy could be written against the Infracost JSON format, not how to run Sentinel, since that's tied to HashiCorp's cloud platform.

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
    env:
      TF_ROOT: examples/terraform-project/code

    steps:
      - uses: actions/checkout@v2

      - name: Setup Infracost
        uses: infracost/actions/setup@v2
        with:
          api-key: ${{ secrets.INFRACOST_API_KEY }}

      - name: Setup Sentinel
        run: |
          mkdir /tmp/sentinel
          curl -o /tmp/sentinel/sentinel.zip https://releases.hashicorp.com/sentinel/0.18.4/sentinel_0.18.4_linux_amd64.zip
          unzip -d /tmp/sentinel /tmp/sentinel/sentinel.zip
          echo "/tmp/sentinel" >> $GITHUB_PATH

      # Checkout the branch you want Infracost to compare costs against. This example is using the
      # target PR branch.
      - name: Checkout base branch
        uses: actions/checkout@v2
        with:
          ref: '${{ github.event.pull_request.base.ref }}'

      # Generate an Infracost cost estimate baseline from the comparison branch, so that Infracost can compare the cost difference.
      - name: Generate Infracost cost estimate baseline
        run: |
          infracost breakdown --path ${TF_ROOT} \
                              --format json \
                              --out-file /tmp/infracost-base.json

      # Checkout the current PR branch so we can create a diff.
      - name: Checkout PR branch
        uses: actions/checkout@v2

      # Generate an Infracost diff and save it to a JSON file.
      - name: Generate Infracost diff
        run: |
          infracost diff --path ${TF_ROOT} \
                              --format json \
                              --compare-to /tmp/infracost-base.json \
                              --out-file /tmp/infracost.json

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
