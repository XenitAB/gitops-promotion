# GitOps Promotion

A tool to do automatic promotion with a GitOps workflow.

## Using with Github

gitops-promotion is available as a Github Action.

Depending on which container registry you are using, you may be able to set up triggers that activates your gitops-promotion workflow. If this is not the case, you can use GitHub [repository_dispatch](https://docs.github.com/en/actions/learn-github-actions/events-that-trigger-workflows#repository_dispatch) events. These allow GitHub actions on one repository to notify another repository. Use the excellent [repository-dispatch GitHub Action](https://github.com/marketplace/actions/repository-dispatch) for readable YAML. You would add a step at the end of your container-building workflow that looks something like this:

```yaml
      - name: Notify gitops-promotion workflow
        uses: peter-evans/repository-dispatch@v1
        with:
          token: ${{ secrets.GITOPS_REPO_TOKEN }}
          repository: my-org/my-gitops
          event-type: image-push
          client-payload: |
            {
              "tag": "${{ github.sha }}"
            }
```
The `repository` parameter holds the repository where you want to run `gitops-promotion`. The normal `${{ secrets.GITHUB_TOKEN }}` only has access to the local repository running in which the workflow is running, so we need to set up and pass an access token (GITOPS_REPO_TOKEN) that has access to that repository.

Here is a complete example GitHub workflow for pushing a containerized app to GitHub Container Registry:

```yaml
on:
  push:
    branches:
      - main

jobs:
  build-app:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v2
      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v1
      - name: Login to GitHub Container Registry
        uses: docker/login-action@v1
        with:
          registry: ghcr.io
          username: ${{ github.repository_owner }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - name: Build and push
        uses: docker/build-push-action@v2
        with:
          push: true
          tags: ghcr.io/${{ github.repository_owner }}/my-app:${{ github.sha }}
      - name: Notify gitops-promotion workflow
        uses: peter-evans/repository-dispatch@v1
        with:
          token: ${{ secrets.GITOPS_REPO_TOKEN }}
          repository: ${{ github.repository_owner }}/my-gitops
          event-type: image-push
          client-payload: |
            {
              "tag": "${{ github.sha }}"
            }
```

In your gitops repository, you can react to `repository-dispatch` events and trigger promotion:

```yaml
on:
  repository_dispatch:
    types:
      - image-push
jobs:
  new-pr:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v2
        with:
          # gitops-promotion currently needs access to history
          fetch-depth: 0
      - uses: xenitab/gitops-promotion@v1
        with:
          action: new
          token: ${{ secrets.GITHUB_TOKEN }}
          group: apps
          app: my-app
          tag: ${{ github.event.client_payload.tag }}
```

This simple example will start the promotion of `my-app` onto the first environment defined in the `gitops-promotion.yaml` file. In order to promote `my-app` to further environments, set up a separate workflow that reacts to merges from previous promotions, like so:

```yaml
on:
  push:
    branches:
      - main
jobs:
  promote-app:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v2
        with:
          # gitops-promotion currently needs access to history
          fetch-depth: 0
      - uses: xenitab/gitops-promotion@v1
        with:
          action: promote
          token: ${{ secrets.GITHUB_TOKEN }}
```

## Building

You will need pkg-config and libgit2, please install it from your package manager.

## Testing the GitHub provider

The test suite for the GitHub provider requires access to an actual GitHub repository. In order to run these tests, create an empty repository and set up an access key and invoke the tests like so:

env GITHUB_URL='' GITHUB_TOKEN='' go test ./...

The GitHub Action CI runs the tests against https://github.com/gitops-promotion/gitops-promotion-testing.
