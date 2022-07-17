#!/bin/bash
set -e

IMAGE=registry.lestak.sh/centaurid
TAG=$GIT_COMMIT

docker buildx build -f devops/docker/centaurid.Dockerfile \
    --platform=linux/arm64,linux/amd64,linux/arm/v7 \
    -t $IMAGE:$TAG \
    --push \
    .

sed "s,$IMAGE:.*,$IMAGE:$TAG,g" devops/k8s/*.yaml | kubectl apply -f -
