# Terragrunt

This example shows how to run Infracost in GitHub Actions with Terragrunt.

[//]: <> (BEGIN EXAMPLE)
```yml
name: Terragrunt project
on: [pull_request]

jobs:
  terragrunt-project:
    name: Terragrunt project
    runs-on: ubuntu-latest
    env:
      TF_ROOT: examples/plan-json/terragrunt/code

    steps:
      - name: Checkout PR branch
        uses: actions/checkout@v2

      - name: Setup Terraform
        uses: hashicorp/setup-terraform@v2
        with:
          terraform_wrapper: false # This is recommended so the `terraform show` command outputs valid JSON

      - name: Setup Terragrunt
        uses: autero1/action-terragrunt@v1.1.0
        with:
          terragrunt_version: 0.37.0

      # Generate plan JSON files for all Terragrunt modules and
      # add them to an Infracost config file
      - name: Generate plan JSONs
        run: |
          terragrunt run-all --terragrunt-ignore-external-dependencies plan -out=plan.cache

          # Find the plan files
          plans=($(find . -name plan.cache | tr '\n' ' '))

          # Generate plan JSON files by running terragrunt show for each plan file
          planjsons=()
          for plan in "${plans[@]}"; do
            # Find the Terraform working directory for running terragrunt show
            # We want to take the dir of the plan file and strip off anything after the .terraform-cache dir
            # to find the location of the Terraform working directory that contains the Terraform code
            dir=$(dirname $plan)
            dir=$(echo "$dir" | sed 's/\(.*\)\/\.terragrunt-cache\/.*/\1/')

            echo "Running terragrunt show for $(basename $plan) for $dir";
            terragrunt show -json $(basename $plan) --terragrunt-working-dir=$dir --terragrunt-no-auto-init > $dir/plan.json
            planjsons=(${planjsons[@]} "$dir/plan.json")
          done

          # Sort the plan JSONs so we get consistent project ordering in the config file
          IFS=$'\n' planjsons=($(sort <<<"${planjsons[*]}"))

          # Generate Infracost config file
          echo -e "version: 0.1\n\nprojects:\n" > infracost.yml
          for planjson in "${planjsons[@]}"; do
            echo -e "  - path: ${TF_ROOT}/$planjson" >> infracost.yml
          done
        working-directory: ${{ env.TF_ROOT }}

      - name: Setup Infracost
        uses: infracost/actions/setup@v2
        # See https://github.com/infracost/actions/tree/master/setup for other inputs
        # If you can't use this action, see Docker images in https://infracost.io/cicd
        with:
          api-key: ${{ secrets.INFRACOST_API_KEY }}

      # Generate an Infracost diff and save it to a JSON file.
      - name: Generate Infracost diff
        run: |
          infracost diff --config-file=${TF_ROOT}/infracost.yml \
                         --format=json \
                         --out-file=/tmp/infracost.json

      # Posts a comment to the PR using the 'update' behavior.
      # This creates a single comment and updates it. The "quietest" option.
      # The other valid behaviors are:
      #   delete-and-new - Delete previous comments and create a new one.
      #   hide-and-new - Minimize previous comments and create a new one.
      #   new - Create a new cost estimate comment on every push.
      # See https://www.infracost.io/docs/features/cli_commands/#comment-on-pull-requests for other options.
      # The INFRACOST_ENABLE_CLOUD​=true section instructs the CLI to send its JSON output to Infracost Cloud.
      #   This SaaS product gives you visibility across all changes in a dashboard. The JSON output does not
      #   contain any cloud credentials or secrets.
      - name: Post Infracost comment
        run: |
          INFRACOST_ENABLE_CLOUD​=true infracost comment github --path=/tmp/infracost.json \
                                   --repo=$GITHUB_REPOSITORY \
                                   --github-token=${{github.token}} \
                                   --pull-request=${{github.event.pull_request.number}} \
                                   --behavior=update
```
[//]: <> (END EXAMPLE)
