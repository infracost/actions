# Infracost Get Comment Action

This GitHub Action retrieves the body of the latest Github comment posted using the `infracost/actions/comment` action. We're still developing the use-cases for this action.

## Usage

The action can be used as follows.

```yml
steps:
  - name: Infracost get comment
    id: get-comment
    uses: infracost/actions/get-comment@v1
  
  - name: Show comment
    run: echo "${{ steps.get-comment.outputs.body }}"
```

## Inputs

The action supports the following inputs:

- `target-type`: Optional. The target-type set when the comment was posted (if any), either `pull_request` or `commit`.

- `tag`: Optional. Customize the comment tag. This is added to the comment as a markdown comment to detect the previously posted comments. This is useful if you have multiple workflows that post comments to the same pull request or commit.

- `github-token`: Optional, default to `${{ github.token }}`. This is the default GitHub token available to actions and is used to get comments.

## Outputs

This action sets the following output:
 
- `body`: The body of the latest matching comment.
