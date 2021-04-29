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
git checkout "$BRANCH" || exit 16
zip data.zip -P "$pass" -r data >/dev/null || exit 10
cd .. || exit 11
if [ -z "$SKIP_BUILD" ]
then
  echo "Building for branch ${BRANCH}"
  cp ../da-ds/uuid.py ../da-ds/gitops.py ../da-ds/detect-removed-commits.sh ./sources/ || exit 14
  if [ -z "$NO_P2O" ]
  then
    docker build -f ./docker-images/Dockerfile -t "${DOCKER_USER}/sync-data-sources-${BRANCH}" --build-arg BRANCH="${BRANCH}" .
    bs=$?
  else
    docker build -f ./docker-images/Dockerfile.nop2o -t "${DOCKER_USER}/sync-data-sources-${BRANCH}" --build-arg BRANCH="${BRANCH}" .
    bs=$?
  fi
  rm -f ./sources/uuid.py ./sources/gitops.py ./sources/detect-removed-commits.sh
  if [ ! "$bs" = "0" ]
  then
    exit 12
  fi
fi

if [ -z "$SKIP_PUSH" ]
then
  echo "Pushing"
  docker push "${DOCKER_USER}/sync-data-sources-${BRANCH}" || exit 13
fi

echo 'OK'

