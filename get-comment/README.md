# Infracost Get Comment Action

This GitHub Action retrieves the body of the latest Github comment posted using the `infracost/actions/comment` action.

## Usage

The action can be used as follows:

```yml
steps:
  - name: Infracost comment
    uses: infracost/actions/get-comment@v1
    with: 
      path: /tmp/infracost.json
```

## Inputs

The action supports the following inputs:

- `target-type`: Optional. The target-type set when the comment was posted (if any), either `pull_request` or `commit`.

- `github-token`: Optional, default to `${{ github.token }}`. This is the default GitHub token available to actions and is used to get comments.

## Outputs

This action sets the following output:
 
- `body`: The body of the latest matching comment.
