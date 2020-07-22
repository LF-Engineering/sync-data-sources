#!/bin/bash
# DOCKER_USER=dajohn [SKIP_BUILD=1] [SKIP_PUSH=1] [PRUNE=1] ./docker-images/build-validate.sh
# DOCKER_USER=dajohn [BRANCH=test|prod] [PRUNE=1] ./docker-images/remove.sh
if [ -z "${DOCKER_USER}" ]
then
  echo "$0: you need to set docker user via DOCKER_USER=username"
  exit 1
fi

if [ ! -z "$PRUNE" ]
then
  docker system prune -f
fi

if [ -z "$SKIP_BUILD" ]
then
  echo "Building"
  docker build -f ./docker-images/Dockerfile.validate -t "${DOCKER_USER}/validate-sync-data-sources" . || exit 2
fi

if [ -z "$SKIP_PUSH" ]
then
  echo "Pushing"
  docker push "${DOCKER_USER}/validate-sync-data-sources" || exit 3
fi

echo 'OK'
