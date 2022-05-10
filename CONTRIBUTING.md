# Contributing

## Adding examples

Copy another example that is similar to what you'd like to add. If you don't know which to pick, try the `terraform-directory`.
1. Update your example's readme file with the required steps.
2. Run `npm run examples:generate_tests` to generate tests for your example.
3. Follow [updating the golden files](#updating-the-golden-files) to update the test output, review the new additions/changes to ensure they are what you expect for your example.
4. Update the [repo readme](readme.md), the [examples readme](examples/readme.md) and your new example's readme with the description of the example.
5. Send a pull request and wait for an Infracost team member to review it.

## Testing examples

Examples are tested by extracting them from the README.md files into a GitHub Actions workflow.

To extract the examples, the `npm run examples:generate_tests` script loops through the example directories, reads the READMEs and extracts any YAML code blocks between the markdown comment markers:

````
[//]: <> (BEGIN EXAMPLE)
```yml
name: My Example
on: [pull_request]

jobs:
  my_example:
    ...
```
[//]: <> (END EXAMPLE)
````

The examples are then modified in two ways:
1. Use local paths for any Infracost actions, e.g. replace `uses: infracost/actions/setup@v2` with `./setup`.
2. Replace any `infracost/actions/comment` steps with steps to generate and test the content of the comment using golden files from the [./testdata](./testdata) directory.

All the examples are then added to the `examples_test` GitHub Actions workflow as separate jobs.

The script that handles extracting and modifying the examples is [./scripts/generateExamplesTest.js](./scripts/generateExamplesTest.js)

### Testing locally

You can test the examples locally with [act](https://github.com/nektos/act). To install on Mac OS X:

```sh
brew install --HEAD act # Use HEAD so we get the artifact server support
```

Install packages:

```sh
npm install
```

Setup required environment variables

```
cp .env.example .env
# Edit .env to add any required environment variables.
```

Then run (with `act`, select the Medium size image):

```sh
npm run examples:test
```

### Updating the golden files

You can update the golden files for the examples by running:

```sh
export GITHUB_TOKEN=<GITHUB_PERSONAL_ACCESS_TOKEN>
npm run examples:update_golden
```

# Releases

One of the Infracost team members should follow these steps to release this repo:
1. `git checkout master && git pull origin master`
2. `git tag vX.Y.Z && git push origin vX.Y.Z` (following semantic versioning)
3. `git tag -f v1 && git push origin v1 -f` (assuming the new release is backward compatible with v1)
4. Confirm that the [repo tags](https://github.com/infracost/actions/tags) show matching git SHAs for vX.Y.X and v1.
5. Create a release for v.X.Y.Z and publish it in the [GitHub Marketplace](https://github.com/marketplace/actions/infracost-actions).
