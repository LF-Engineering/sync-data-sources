#!/bin/bash
if [ -z "${DOCKER_USER}" ]
then
  echo "$0: you need to set docker user via DOCKER_USER=username"
  exit 2
fi
if [ -z "${BRANCH}" ]
then
  echo "$0: you need to set dev-analytics-branch via BRANCH=test|prod"
  exit 2
fi
command="$1"
if [ -z "${command}" ]
then
  export command=/bin/bash
fi
docker run -it "${DOCKER_USER}/sync-data-sources-${BRANCH}" "$command"