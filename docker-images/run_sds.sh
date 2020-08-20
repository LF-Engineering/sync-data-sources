#!/bin/bash
# DRY=1 - run in dry-run mode
# SH=1 - execute /bin/bash instead of sds
# DBG=1 - output docker command
# NO=1 - do not run actual docker command (can be used as DBG=1 NO=1)
# DM=1 - run in daemon mode, only applicable when SH is not set
if [ -z "${1}" ]
then
  echo "$0: you need to specify env as a 1st argument: test|prod"
  exit 1
fi
if [ ! -z "${DRY}" ]
then
  #export SDS_ST=1
  #export SDS_NCPUS=16
  #export SDS_NCPUS_SCALE=4
  #export SDS_DEBUG=1
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
  export SDS_SKIP_SH=1
  export SDS_SKIP_DATA=1
  export SDS_SKIP_AFFS=1
  export SDS_SKIP_ALIASES=1
  export SDS_SKIP_DROP_UNUSED=1
  export SDS_NO_INDEX_DROP=1
  export SDS_SKIP_CHECK_FREQ=1
  export SDS_SKIP_ES_DATA=1
  export SDS_SKIP_ES_LOG=1
  export SDS_SKIP_DEDUP=1
  export SDS_SKIP_EXTERNAL=1
  export SDS_SKIP_PROJECT=1
  export SDS_SKIP_PROJECT_TS=1
  export SDS_SKIP_SYNC_INFO=1
  export SDS_SKIP_VALIDATE_GITHUB_API=1
  export SDS_SKIP_SSAW=1
  export SDS_SKIP_SORT_DURATION=1
  export SDS_SKIP_MERGE=1
  export SDS_SKIP_HIDE_EMAILS=1
  export SDS_SKIP_ORG_MAP=1
  export SDS_SKIP_ENRICH_DS=1
  export SDS_SKIP_COPY_FROM=1
  export SDS_SKIP_P2O=1
  export SDS_DRY_RUN=1
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
  #export SDS_DRY_RUN_ALLOW_ORG_MAP=1
  #export SDS_DRY_RUN_ALLOW_ENRICH_DS=1
  #export SDS_DRY_RUN_ALLOW_DET_AFF_RANGE=1
  #export SDS_DRY_RUN_ALLOW_COPY_FROM=1
  #export SDS_DRY_RUN_ALLOW_SSAW=1
  #export SDS_ONLY_VALIDATE=1
  #export SDS_ONLY_P2O=1
fi
envstr="-e BRANCH=\"$1\""
for renv in SDS_SSAW_URL SH_USER SH_HOST SH_PORT SH_PASS SH_DB SDS_ES_URL SDS_GITHUB_OAUTH AUTH0_URL AUTH0_AUDIENCE AUTH0_CLIENT_ID AUTH0_CLIENT_SECRET AFFILIATION_API_URL METRICS_API_URL REPO_ACCESS
do
  if ( [ "${renv}" = "SDS_SSAW_URL" ] || [ "${renv}" = "SDS_ES_URL" ]  || [ "${renv}" = "SDS_GITHUB_OAUTH" ] )
  then
    fn="helm-charts/sds-helm/sds-helm/secrets/${renv:4}.${1}.secret"
    fn2="helm-charts/sds-helm/sds-helm/secrets/${renv:4}.secret"
  else
    fn="helm-charts/sds-helm/sds-helm/secrets/${renv}.${1}.secret"
    fn2="helm-charts/sds-helm/sds-helm/secrets/${renv}.secret"
  fi
  renvval="`cat ${fn} 2>/dev/null`"
  if [ -z "${renvval}" ]
  then
    renvval="`cat ${fn2} 2>/dev/null`"
  fi
  if [ -z "${renvval}" ]
  then
    echo "Cannot get value for ${renv} env variable from ${fn}/${fn2} files"
    exit 1
  fi
  envstr="${envstr} -e ${renv}=\"${renvval}\""
done
for uenv in SSO_API_KEY SSO_API_SECRET SSO_AUDIENCE SSO_USER_SERVICE USER_SERVICE_URL
do
  secret_value=`cat helm-charts/sds-helm/sds-helm/secrets/${uenv}.prod.secret`
  envstr="${envstr} -e ${uenv}=\"${secret_value}\""
done
envs=`env | grep SDS_ | sort`
for env in ${envs}
do
  OIFS=${IFS}
  IFS='='
  a=(${env})
  IFS=${OIFS}
  envstr="${envstr} -e ${a}=\"${!a}\""
done
if [ -z "${2}" ]
then
  flg="-v /root/.perceval:/root/.perceval"
else
  flg="-v /root/.perceval:/root/.perceval -v /data/${2}:/root"
fi
if [ -z "${SH}" ]
then
  cmd="/run.sh"
  if [ ! -z "${DM}" ]
  then
    flg="${flg} -d"
  fi
else
  cmd="/bin/bash"
  flg="${flg} -it"
fi
cmd="docker run ${envstr} ${flg} \"dajohn/sync-data-sources-${1}:latest\" \"${cmd}\""
if [ ! -z "${DBG}" ]
then
  echo $cmd
fi
if [ -z "${NO}" ]
then
  docker pull "dajohn/sync-data-sources-${1}:latest"
  eval $cmd
fi
