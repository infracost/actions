# Private Terraform module

This example shows how to run Infracost in GitHub Actions with a Terraform project that uses a private Terraform module. This requires a secret to be added to your GitHub repository called `GIT_SSH_KEY` containing a private key so that Infracost can access the private repository.

[//]: <> (BEGIN EXAMPLE)
```yml
name: Private Terraform module
on: [pull_request]

env:
  # Use the same ssh-agent socket value across all jobs
  # Useful when a GH action is using SSH behind-the-scenes
  SSH_AUTH_SOCK: /tmp/ssh_agent.sock
  
jobs:
  private-terraform-module:
    name: Private Terraform module
    runs-on: ubuntu-latest
    env:
      TF_ROOT: examples/private-terraform-module/code

    steps:
      # Add your git SSH key so Infracost can checkout the private modules
      - name: add GIT_SSH_KEY
        run: |
          ssh-agent -a $SSH_AUTH_SOCK
          echo "${{ secrets.GIT_SSH_KEY }}" | tr -d '\r' | ssh-add -

      # Checkout the base branch of the pull request (e.g. main/master).
      - name: Checkout base branch
        uses: actions/checkout@v2
        with:
          ref: '${{ github.event.pull_request.base.ref }}'
          
      - name: Setup Infracost
        uses: infracost/actions/setup@v2
        # See https://github.com/infracost/actions/tree/master/setup for other inputs
        # If you can't use this action, see Docker images in https://infracost.io/cicd
        with:
          api-key: ${{ secrets.INFRACOST_API_KEY }}

      # Generate Infracost JSON file as the baseline.
      - name: Generate Infracost cost estimate baseline
        run: |
          infracost breakdown --path=${TF_ROOT} \
                              --format=json \
                              --out-file=/tmp/infracost-base.json

      # Checkout the current PR branch so we can create a diff.
      - name: Checkout PR branch
        uses: actions/checkout@v2

      # Generate an Infracost diff and save it to a JSON file.
      - name: Generate Infracost diff
        run: |
          infracost diff --path=${TF_ROOT} \
                              --format=json \
                              --compare-to=/tmp/infracost-base.json \
                              --out-file=/tmp/infracost.json

      # Posts a comment to the PR using the 'update' behavior.
      # This creates a single comment and updates it. The "quietest" option.
      # The other valid behaviors are:
      #   delete-and-new - Delete previous comments and create a new one.
      #   hide-and-new - Minimize previous comments and create a new one.
      #   new - Create a new cost estimate comment on every push.
      # See https://www.infracost.io/docs/features/cli_commands/#comment-on-pull-requests for other options.
      - name: Post Infracost comment
        run: |
          infracost comment github --path=/tmp/infracost.json \
                                   --repo=$GITHUB_REPOSITORY \
                                   --github-token=${{github.token}} \
                                   --pull-request=${{github.event.pull_request.number}} \
                                   --behavior=update
```
[//]: <> (END EXAMPLE)
