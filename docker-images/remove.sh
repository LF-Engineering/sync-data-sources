#!/bin/bash
# DOCKER_USER=lukaszgryglicki SKIP_BUILD=1 SKIP_PUSH=1 BRANCH=test|prod ./docker-images/build.sh
# DOCKER_USER=lukaszgryglicki BRANCH=test|prod PRUNE=1 ./docker-images/remove.sh
if [ -z "${DOCKER_USER}" ]
then
  echo "$0: you need to set docker user via DOCKER_USER=username"
  exit 1
fi
if [ -z "${BRANCH}" ]
then
  echo "$0: you need to set dev-analytics-branch via BRANCH=test|prod"
  exit 2
fi
rm -rf ./sources/data || exit 1
docker image rm -f "${DOCKER_USER}/sync-data-sources-${BRANCH}" || exit 2
if [ ! -z "$PRUNE" ]
then
  docker system prune -f
fi
