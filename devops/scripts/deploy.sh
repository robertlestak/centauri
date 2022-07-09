#!/bin/bash
set -e

IMAGE=registry.lestak.sh/centaurid
TAG=$GIT_COMMIT

docker build -f devops/docker/centaurid.Dockerfile \
    -t $IMAGE:$TAG \
    .

docker push $IMAGE:$TAG

sed "s,$IMAGE:.*,$IMAGE:$TAG,g" devops/k8s/*.yaml | kubectl apply -f -
