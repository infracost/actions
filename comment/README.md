# Infracost Comment Action

This GitHub Action takes `infracost breakdown` JSON output and posts it as a GitHub comment. It assumes the `infracost` binary has already been installed using the [setup](../setup) action.

## Usage

The action can be used as follows:

```yml
steps:
  - name: Infracost comment
    uses: infracost/actions/comment@v1
    with: 
      path: /tmp/infracost.json
```

## Inputs

The action supports the following inputs:

- `path`: Required. The path to the `infracost breakdown` JSON that will be passed to `infracost output`. For multiple paths, pass a glob pattern (e.g. "infracost_*.json", glob needs quotes) or a JSON array of paths.

- `behavior`: Optional, defaults to `update`. The behavior to use when posting cost estimate comments. Must be one of the following:  
  - `update`:  Use a single comment to display cost estimates, creating one if none exist. The GitHub comments UI can be used to see when/what has changed when a comment is updated. Pull request followers will only be notified on the comment create (not update), and the comment will stay at the same location in the comment history.
  - `delete_and_new`: Delete previous cost estimate comments and create a new one. Pull request followers will be notified on each comment.
  - `hide_and_new`: Minimize previous cost estimate comments and create a new one. Pull request followers will be notified on each comment.
  - `new`:  Create a new cost estimate comment. Pull request followers will be notified on each comment.

- `targetType`: Optional. Which objects should be commented on, either `pr` (for pull requests) or `commit`.

- `GITHUB_TOKEN`: Optional, default to `${{ github.token }}`. This is the default GitHub token available to actions and is used to post comments.

## Outputs

This action does not set any direct outputs.
