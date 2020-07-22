#!/bin/bash
# DOCKER_USER=dajohn BRANCH=test|prod [SKIP_PULL=1] [SKIP_BUILD=1] [SKIP_PUSH=1] [PRUNE=1] [API_REPO_PATH="$HOME/dev/LF/dev-analytics-api"] ./docker-images/build.sh
# DOCKER_USER=dajohn BRANCH=test|prod [PRUNE=1] ./docker-images/remove.sh
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

if [ -z "${API_REPO_PATH}" ]
then
  API_REPO_PATH="$HOME/dev/LF-Engineering/dev-analytics-api"
fi

if [ ! -z "$PRUNE" ]
then
  docker system prune -f
fi

pass=`cat zippass.secret`
if [ -z "$pass" ]
then
  echo "$0: you need to specify ZIP password in gitignored file zippass.secret"
  exit 3
fi

cwd="`pwd`"
cd $API_REPO_PATH || exit 4
git checkout "$BRANCH" || exit 5
if [ -z "$SKIP_PULL" ]
then
  git pull || exit 6
fi
rm -rf "$cwd/sources/data" "$cwd/sources/data.zip" || exit 7
cp -R app/services/lf/bootstrap/fixtures/ "$cwd/sources/data" || exit 8
cd "${cwd}/sources" || exit 9
zip data.zip -P "$pass" -r data >/dev/null || exit 10
cd .. || exit 11

if [ -z "$SKIP_BUILD" ]
then
  echo "Building"
  docker build -f ./docker-images/Dockerfile -t "${DOCKER_USER}/sync-data-sources-${BRANCH}" . || exit 12
fi

if [ -z "$SKIP_PUSH" ]
then
  echo "Pushing"
  docker push "${DOCKER_USER}/sync-data-sources-${BRANCH}" || exit 13
fi

echo 'OK'

