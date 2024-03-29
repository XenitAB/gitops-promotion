#!/bin/sh

set -e

case $ACTION in
    new)
        /usr/local/bin/gitops-promotion new \
            --provider github \
            --sourcedir "$GITHUB_WORKSPACE" \
            --token "$TOKEN" \
            --group "$GROUP" \
            --app "$APP" \
            --tag "$TAG"
        ;;
    feature)
        /usr/local/bin/gitops-promotion feature \
            --provider github \
            --sourcedir "$GITHUB_WORKSPACE" \
            --token "$TOKEN" \
            --group "$GROUP" \
            --app "$APP" \
            --tag "$TAG" \
            --feature "$FEATURE"
        ;;
    feature-stale)
        /usr/local/bin/gitops-promotion feature-stale \
            --provider github \
            --sourcedir "$GITHUB_WORKSPACE" \
            --token "$TOKEN" \
        ;;
    promote)
        /usr/local/bin/gitops-promotion promote \
            --provider github \
            --sourcedir "$GITHUB_WORKSPACE" \
            --token "$TOKEN"
        ;;
    status)
        /usr/local/bin/gitops-promotion status \
            --provider github \
            --sourcedir "$GITHUB_WORKSPACE" \
            --token "$TOKEN"
        ;;
    *)
        echo "Unkown action $ACTION"
        exit 1
esac
