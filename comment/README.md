# Infracost Comment Action

This GitHub Action takes `infracost breakdown` JSON output and posts it as a GitHub comment. It assumes the `infracost` binary has already been installed using the [setup](../setup) action. It uses the default GitHub token, which is available for actions, to post comments (you can override it with inputs). This action uses the [compost](https://github.com/infracost/compost) CLI tool internally.

## Usage

The action can be used as follows. See the [top-level readme](https://github.com/infracost/actions) for examples of how this actions can be combined with the setup action to run Infracost.

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
  - `update`: Create a single comment and update it on changes. This is the "quietest" option. The GitHub comments UI shows what/when changed when the comment is updated. Pull request followers will only be notified on the comment create (not updates), and the comment will stay at the same location in the comment history.
  - `delete-and-new`: Delete previous cost estimate comments and create a new one. Pull request followers will be notified on each comment.
  - `hide-and-new`: Minimize previous cost estimate comments and create a new one. Pull request followers will be notified on each comment.
  - `new`: Create a new cost estimate comment. Pull request followers will be notified on each comment.

- `target-type`: Optional. Which objects should be commented on, either `pull-request` or `commit`.

- `tag`: Optional. Customize the comment tag. This is added to the comment as a markdown comment (hidden) to detect the previously posted comments. This is useful if you have multiple workflows that post comments to the same pull request or commit.

- `github-token`: Optional, default to `${{ github.token }}`. This is the default GitHub token available to actions and is used to post comments. The default [token permissions](https://docs.github.com/en/actions/learn-github-actions/workflow-syntax-for-github-actions#permissions) work fine; `pull-requests: write` is required if you need to customize these.

    ```yml
    steps:
      - name: Infracost comment
        uses: infracost/actions/comment@v1
        with:
          ...
        permissions:
          pull-requests: write
    ```

## Outputs

This action sets the following output:

- `body`: The body of comment that was posted.

