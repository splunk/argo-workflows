#!/bin/bash

set -e

TAG=$1
TIMESTAMP=$(date +%Y%m%d%H%M%S)
IMAGE_URL="hack.splunk.io"
IMAGE_NAMESPACE="$IMAGE_URL" VERSION="$TAG-$TIMESTAMP" make workflow-controller-image

IMAGE="$IMAGE_URL/workflow-controller:$TAG-$TIMESTAMP"
kind load docker-image "${IMAGE}" --name hack-2025-03-cluster

echo "--------------------------------"
echo "Updating deployment api with image $IMAGE..."
kubectl patch deployment argo-workflows-workflow-controller \
  -n default \
  --type='json' \
  -p='[{"op": "replace", "path": "/spec/template/spec/containers/0/imagePullPolicy", "value": "Never"}]'
kubectl set image deployment/argo-workflows-workflow-controller controller=$IMAGE -n default
