# Cost Threshholds Example

This example shows how to set thresholds that limit when a comment is posted.  For simplicity, this is based off the terraform-plan-json example which does not require Terraform to be installed.

[//]: <> (BEGIN EXAMPLE)
```yml
name: Thresholds
on: [pull_request]

jobs:
  terraform-plan-json:
    name: Thresholds
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v2
      
      - name: Setup Infracost
        uses: infracost/actions/setup@v1
        with:
          api_key: ${{ secrets.INFRACOST_API_KEY }}
          
      - name: Run Infracost
        run: infracost breakdown --path=examples/terraform-plan-json/code/plan.json --format=json --out-file=/tmp/infracost.json.json
        
      - name: Calculate Cost Change
        id: cost-change
        uses: actions/github-script@v5
        with:
          script: |
            // Read the breakdown JSON and get the past and current total monthly costs
            const breakdown = require('/tmp/infracost.json.json');
            const past = breakdown.pastTotalMonthlyCost;
            const current = breakdown.totalMonthlyCost;
            
            // Calculate the percent and $ diffs
            let absolutePercentChange = 0;
            let absoluteCostChange = 0         ;   
            if (past != 0) {
              absolutePercentChange = 100 * Math.abs((current - past) / past);
            }            
            absoluteCostChange = Math.abs(past - current);
            
            // Set the calculated diffs as outputs to be used in future steps
            core.setOutput('absolute-percent-change', absolutePercentChange);
            core.setOutput('absolute-cost-change', absoluteCostChange);
      - name: Post the comment
        uses: infracost/actions/comment@v1
        if: ${{ steps.cost-change.outputs.absolute-percent-change > 1 }} # Only comment if cost changed by more than 1%
        # if: ${{ steps.cost-change.outputs.absolute-cost-change > 100 }} # Only comment if cost changed by more than $100 
        with:
          path: /tmp/infracost.json.json
```
[//]: <> (END EXAMPLE)
