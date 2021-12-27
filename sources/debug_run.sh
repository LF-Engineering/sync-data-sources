#!/bin/bash
# Example: SH_HOST=127.0.0.1 SH_PORT=13306 SH_DB=sortinghat SH_USER=sortinghat SH_PASS=pwd ./debug_run.sh test
if [ -z "$1" ]
then
  echo "$0: you need to specify env: test, prod"
  exit 1
fi
# TODO FIXME: when you try to use this to run the actuall commands
# please almost always use SDS_ONLY_P2O=1 when not running in dry run mode
export DA_AFFS_API_FAIL_FATAL=1
#export SDS_ST=1
#export SDS_NCPUS=16
#export SDS_NCPUS_SCALE=4
export SDS_DEBUG=2
export SDS_CMDDEBUG=2
#export SDS_FIXTURES_RE=''
#export SDS_DATASOURCES_RE=''
#export SDS_PROJECTS_RE=''
#export SDS_ENDPOINTS_RE=''
#export SDS_TASKS_RE=''
#export SDS_FIXTURES_SKIP_RE=''
#export SDS_DATASOURCES_SKIP_RE=''
#export SDS_PROJECTS_SKIP_RE=''
#export SDS_ENDPOINTS_SKIP_RE=''
#export SDS_TASKS_SKIP_RE=''
#export SDS_SKIPTIME=1
#export SDS_SKIP_REENRICH="jira,gerrit,confluence,bugzilla"
#export SDS_SKIP_SH=1
#export SDS_SKIP_DATA=1
#export SDS_SKIP_AFFS=1
export SDS_SKIP_ALIASES=1
export SDS_SKIP_DROP_UNUSED=1
export SDS_NO_INDEX_DROP=1
export SDS_SKIP_CHECK_FREQ=1
export SDS_SKIP_ES_DATA=1
export SDS_SKIP_ES_LOG=1
export SDS_SKIP_DEDUP=1
export SDS_SKIP_EXTERNAL=1
#export SDS_SKIP_PROJECT=1
#export SDS_SKIP_PROJECT_TS=1
export SDS_SKIP_SYNC_INFO=1
#export SDS_SKIP_VALIDATE_GITHUB_API=1
export SDS_SKIP_SORT_DURATION=1
export SDS_SKIP_MERGE=1
export SDS_SKIP_HIDE_EMAILS=1
export SDS_SKIP_METADATA=1
export SDS_SKIP_CACHE_TOP_CONTRIBUTORS=1
export SDS_SKIP_ORG_MAP=1
export SDS_SKIP_ENRICH_DS=1
export SDS_SKIP_COPY_FROM=1
#export SDS_SKIP_P2O=1
#export SDS_DRY_RUN=1
#export SDS_RUN_DET_AFF_RANGE=1
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
#export SDS_DRY_RUN_ALLOW_MERGE=1
#export SDS_DRY_RUN_ALLOW_HIDE_EMAILS=1
#export SDS_DRY_RUN_ALLOW_METADATA=1
#export SDS_DRY_RUN_ALLOW_CACHE_TOP_CONTRIBUTORS=1
#export SDS_DRY_RUN_ALLOW_ORG_MAP=1
#export SDS_DRY_RUN_ALLOW_ENRICH_DS=1
#export SDS_DRY_RUN_ALLOW_DET_AFF_RANGE=1
#export SDS_DRY_RUN_ALLOW_COPY_FROM=1
#export SDS_ONLY_VALIDATE=1
export SDS_ONLY_P2O=1
if [ -z "${SDS_ES_URL}" ]
then
  export SDS_ES_URL=`cat ../helm-charts/sds-helm/sds-helm/secrets/ES_URL.$1.secret`
fi
if [ -z "${SH_HOST}" ]
then
  export SH_HOST=`cat ../helm-charts/sds-helm/sds-helm/secrets/SH_HOST.$1.secret`
fi
if [ -z "${SH_PORT}" ]
then
  export SH_PORT=`cat ../helm-charts/sds-helm/sds-helm/secrets/SH_PORT.$1.secret`
fi
if [ -z "${SH_DB}" ]
then
  export SH_DB=`cat ../helm-charts/sds-helm/sds-helm/secrets/SH_DB.$1.secret`
fi
if [ -z "${SH_USER}" ]
then
  export SH_USER=`cat ../helm-charts/sds-helm/sds-helm/secrets/SH_USER.$1.secret`
fi
if [ -z "${SH_PASS}" ]
then
  export SH_PASS=`cat ../helm-charts/sds-helm/sds-helm/secrets/SH_PASS.$1.secret`
fi
if [ -z "${AUTH0_URL}" ]
then
  export AUTH0_URL=`cat ../helm-charts/sds-helm/sds-helm/secrets/AUTH0_URL.$1.secret`
fi
if [ -z "${AUTH0_AUDIENCE}" ]
then
  export AUTH0_AUDIENCE=`cat ../helm-charts/sds-helm/sds-helm/secrets/AUTH0_AUDIENCE.$1.secret`
fi
if [ -z "${AUTH0_CLIENT_ID}" ]
then
  export AUTH0_CLIENT_ID=`cat ../helm-charts/sds-helm/sds-helm/secrets/AUTH0_CLIENT_ID.$1.secret`
fi
if [ -z "${AUTH0_CLIENT_SECRET}" ]
then
  export AUTH0_CLIENT_SECRET=`cat ../helm-charts/sds-helm/sds-helm/secrets/AUTH0_CLIENT_SECRET.$1.secret`
fi
if [ -z "${AFFILIATION_API_URL}" ]
then
  export AFFILIATION_API_URL=`cat ../helm-charts/sds-helm/sds-helm/secrets/AFFILIATION_API_URL.$1.secret`
fi
if [ -z "${AUTH0_DATA}" ]
then
  export AUTH0_DATA=`cat ../helm-charts/sds-helm/sds-helm/secrets/AUTH0_DATA.$1.secret`
fi
if [ -z "${METRICS_API_URL}" ]
then
  export METRICS_API_URL=`cat ../helm-charts/sds-helm/sds-helm/secrets/METRICS_API_URL.$1.secret`
fi
if [ -z "${SDS_GITHUB_OAUTH}" ]
then
  export SDS_GITHUB_OAUTH="`cat /etc/github/oauths`"
fi
if ( [ -z "${JWT_TOKEN}" ] && [ -f "token.secret" ] )
then
  export JWT_TOKEN=`cat token.secret`
fi
env | grep SDS_ | sort
env | grep SH_ | sort
./syncdatasources
