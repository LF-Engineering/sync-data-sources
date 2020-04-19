#!/bin/bash
# REPO_ACCESS=https://github_id:github_oauth2_token@github.com/LF-Engineering/dev-analytics-api - use dynamic fixture fetching from repo instead of hardcoded in data.zip
if [ -z "${DOCKER_USER}" ]
then
  echo "$0: you need to set docker user via DOCKER_USER=username"
  exit 2
fi
if [ -z "${BRANCH}" ]
then
  echo "$0: you need to set dev-analytics-branch via BRANCH=commit_SHA"
  exit 2
fi
command="$1"
if [ -z "${command}" ]
then
  export command=/bin/bash
fi
docker run -e SDS_ONLY_VALIDATE=1 -e BRANCH="$BRANCH" -e REPO_ACCESS="$REPO_ACCESS" -it "${DOCKER_USER}/validate-sync-data-sources" "$command"
