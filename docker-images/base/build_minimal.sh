#!/bin/bash
if [ -z "${DOCKER_USER}" ]
then
  export DOCKER_USER=lukaszgryglicki
fi
./collect_and_build.sh && docker build -f Dockerfile -t "${DOCKER_USER}/dev-analytics-grimoire-docker-minimal" . && docker push "${DOCKER_USER}/dev-analytics-grimoire-docker-minimal"
