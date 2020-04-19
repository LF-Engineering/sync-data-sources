#!/bin/bash
# EXEC - start SDS shell, so you can for example ./run.sh manually
if [ -z "$1" ]
then
  echo "$0: you need to specify commit SHA or branch name: test, prod, 01527ed...ab35"
  exit 1
fi
cmd='./run.sh'
if [ ! -z "$2" ]
then
  cmd="$2"
fi
if [ ! -z "$EXEC" ]
then
  docker run -e SDS_ONLY_VALIDATE=1 -e BRANCH="$1" -e SDS_GITHUB_OAUTH=`cat /etc/github/oauths` -e REPO_ACCESS=`cat repo_access.secret` --rm -i -t "lukaszgryglicki/validate-sync-data-sources" /bin/bash
else
  docker run -e SDS_ONLY_VALIDATE=1 -e BRANCH="$1" -e SDS_GITHUB_OAUTH=`cat /etc/github/oauths` -e REPO_ACCESS=`cat repo_access.secret` --rm -i -t "lukaszgryglicki/validate-sync-data-sources" "${cmd}"
fi
