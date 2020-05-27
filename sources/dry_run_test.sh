#!/bin/bash
if [ -z "$1" ]
then
  echo "$0: you need to specify env: test, prod"
  exit 1
fi
#export SDS_ST=1
#export SDS_NCPUS=16
#export SDS_NCPUS_SCALE=4
#export SDS_DEBUG=1
#export SDS_FIXTURES_RE=''
#export SDS_DATASOURCES_RE=''
#export SDS_PROJECTS_RE=''
#export SDS_ENDPOINTS_RE=''
#export SDS_SKIPTIME=1
#export SDS_SKIP_SH=1
#export SDS_SKIP_DATA=1
#export SDS_SKIP_AFFS=1
#export SDS_SKIP_ALIASES=1
###export SDS_SKIP_DROP_UNUSED=1
#export SDS_SKIP_CHECK_FREQ=1
export SDS_SKIP_ES_DATA=1
export SDS_SKIP_ES_LOG=1
#export SDS_SKIP_DEDUP=1
#export SDS_SKIP_EXTERNAL=1
#export SDS_SKIP_PROJECT=1
#export SDS_SKIP_PROJECT_TS=1
#export SDS_SKIP_SYNC_INFO=1
#export SDS_SKIP_VALIDATE_GITHUB_API=1
#export SDS_SKIP_SSAW=1
#export SDS_SKIP_SORT_DURATION=1
export SDS_DRY_RUN=1
#export SDS_DRY_RUN_CODE=3
#export SDS_DRY_RUN_CODE_RANDOM=1
#export SDS_DRY_RUN_SECONDS=1
#export SDS_DRY_RUN_SECONDS_RANDOM=3
#export SDS_DRY_RUN_ALLOW_SSH=1
#export SDS_DRY_RUN_ALLOW_FREQ=1
#export SDS_DRY_RUN_ALLOW_MTX=1
#export SDS_DRY_RUN_ALLOW_RENAME=1
#export SDS_DRY_RUN_ALLOW_ORIGINS=1
#export SDS_DRY_RUN_ALLOW_DEDUP=1
#export SDS_DRY_RUN_ALLOW_PROJECT=1
#export SDS_DRY_RUN_ALLOW_SYNC_INFO=1
#export SDS_DRY_RUN_ALLOW_SORT_DURATION=1
#export SDS_DRY_RUN_ALLOW_SSAW=1
#export SDS_ONLY_VALIDATE=1
export SDS_ES_URL=`cat ../helm-charts/sds-helm/sds-helm/secrets/ES_URL.$1.secret`
export SDS_SSAW_URL=`cat ../helm-charts/sds-helm/sds-helm/secrets/SSAW_URL.$1.secret`
export SH_HOST=`cat ../helm-charts/sds-helm/sds-helm/secrets/SH_HOST.$1.secret`
export SH_PORT=`cat ../helm-charts/sds-helm/sds-helm/secrets/SH_PORT.$1.secret`
export SH_DB=`cat ../helm-charts/sds-helm/sds-helm/secrets/SH_DB.$1.secret`
export SH_USER=`cat ../helm-charts/sds-helm/sds-helm/secrets/SH_USER.$1.secret`
export SH_PASS=`cat ../helm-charts/sds-helm/sds-helm/secrets/SH_PASS.$1.secret`
export SDS_GITHUB_OAUTH="`cat /etc/github/oauths`"
./syncdatasources