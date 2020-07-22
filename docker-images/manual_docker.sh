#!/bin/bash
# EXEC - start SDS shell, so you can for example ./run.sh manually
if [ -z "$1" ]
then
  echo "$0: you need to specify env: test, dev, stg, prod"
  exit 1
fi
cmd='./run.sh'
if [ ! -z "$2" ]
then
  cmd="$2"
fi
if [ ! -z "$EXEC" ]
then
  docker run -e BRANCH="$1" -e SDS_ES_URL=`cat helm-charts/sds-helm/sds-helm/secrets/ES_URL.$1.secret` -e SH_HOST=`cat helm-charts/sds-helm/sds-helm/secrets/SH_HOST.$1.secret` -e SH_PORT=`cat helm-charts/sds-helm/sds-helm/secrets/SH_PORT.$1.secret` -e SH_DB=`cat helm-charts/sds-helm/sds-helm/secrets/SH_DB.$1.secret` -e SH_USER=`cat helm-charts/sds-helm/sds-helm/secrets/SH_USER.$1.secret` -e SH_PASS=`cat helm-charts/sds-helm/sds-helm/secrets/SH_PASS.$1.secret` -e SDS_GITHUB_OAUTH=`cat /etc/github/oauths` -e REPO_ACCESS=`cat repo_access.secret` --rm -i -t -v /root/.perceval:/root/.perceval "dajohn/sync-data-sources-${1}" /bin/bash
else
  docker run -e BRANCH="$1" -e SDS_ES_URL=`cat helm-charts/sds-helm/sds-helm/secrets/ES_URL.$1.secret` -e SH_HOST=`cat helm-charts/sds-helm/sds-helm/secrets/SH_HOST.$1.secret` -e SH_PORT=`cat helm-charts/sds-helm/sds-helm/secrets/SH_PORT.$1.secret` -e SH_DB=`cat helm-charts/sds-helm/sds-helm/secrets/SH_DB.$1.secret` -e SH_USER=`cat helm-charts/sds-helm/sds-helm/secrets/SH_USER.$1.secret` -e SH_PASS=`cat helm-charts/sds-helm/sds-helm/secrets/SH_PASS.$1.secret` -e SDS_GITHUB_OAUTH=`cat /etc/github/oauths` -e REPO_ACCESS=`cat repo_access.secret` --rm -i -v /root/.perceval:/root/.perceval "dajohn/sync-data-sources-${1}" "$cmd"
fi
