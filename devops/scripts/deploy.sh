#!/bin/bash
set -e

BUILDER=b-`uuidgen`

cleanup() {
  docker buildx rm $BUILDER
}

trap cleanup EXIT

docker buildx create --use --name $BUILDER
docker buildx inspect --bootstrap

IMAGE=registry.lestak.sh/centaurid
TAG=$GIT_COMMIT

docker buildx build -f devops/docker/centaurid.Dockerfile \
    --platform=linux/arm64,linux/amd64,linux/arm/v7 \
    -t $IMAGE:$TAG \
    --push \
    .

sed "s,$IMAGE:.*,$IMAGE:$TAG,g" devops/k8s/*.yaml | kubectl apply -f -
