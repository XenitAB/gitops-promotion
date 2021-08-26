# GitOps Promotion

A tool to do automatic promotion with a GitOps workflow.

## Building

You will need pkg-config and libgit2, please install it from your package manager.

## Testing the GitHub provider

The test suite for the GitHub provider requires access to an actual GitHub repository. In order to run these tests, create an empty repository and set up an access key and invoke the tests like so:

env GITHUB_URL='' GITHUB_TOKEN='' go test ./...

The GitHub Action CI runs the tests against https://github.com/gitops-promotion/gitops-promotion-testing.