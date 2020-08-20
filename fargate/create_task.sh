#!/bin/bash
# DRY=1 - run in dry run mode (no task is created, JSON is displayed instead)
# DRYSDS=1 - run SDS in dry run mode (so task will be created but SDS from that task will run in dry run mode)
# AP=1 - use version that uses access points
if [ -z "${AWS_PROFILE}" ]
then
  echo "${0}: you need to specify AWS_PROFILE=..."
  exit 1
fi
if [ -z "${AWS_REGION}" ]
then
  echo "${0}: you need to specify AWS_REGION=..."
  exit 2
fi
if [ -z "${1}" ]
then
  echo "${0}: you need to specify env as a 1st arg: prod|test"
  exit 3
fi
if [ ! -z "${DRYSDS}" ]
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
export SDS_BRANCH="${1}"
if [ -z "${SDS_COMMAND}" ]
then
  export SDS_COMMAND='/fargate_run.sh'
fi
envlist='SDS_ROLE_ARN SDS_FS_ID SDS_TASK_NAME SDS_SSAW_URL SDS_SH_USER SDS_SH_HOST SDS_SH_PORT SDS_SH_PASS SDS_SH_DB SDS_ES_URL SDS_GITHUB_OAUTH SDS_AFFILIATION_API_URL SDS_METRICS_API_URL SDS_AUTH0_URL SDS_AUTH0_AUDIENCE SDS_AUTH0_CLIENT_ID SDS_AUTH0_CLIENT_SECRET SDS_ZIPPASS SDS_REPO_ACCESS'
if [ ! -z "${AP}" ]
then
  envlist="${envlist} SDS_FSAP_ID"
fi
for renv in ${envlist}
do
  if [ -z "${!renv}" ]
  then
    v=`cat "helm-charts/sds-helm/sds-helm/secrets/${renv}.${1}.secret" 2>/dev/null`
    if [ -z "${v}" ]
    then
      v=`cat "helm-charts/sds-helm/sds-helm/secrets/${renv}.secret" 2>/dev/null`
    fi
    if [ -z "${v}" ]
    then
      rrenv=${renv/SDS_/}
      v=`cat "helm-charts/sds-helm/sds-helm/secrets/${rrenv}.${1}.secret" 2>/dev/null`
    fi
    if [ -z "${v}" ]
    then
      v=`cat "helm-charts/sds-helm/sds-helm/secrets/${rrenv}.secret" 2>/dev/null`
    fi
    if [ -z "${v}" ]
    then
      echo "Missing ${renv} environment variable and unable to get it from the secret file"
      exit 4
    fi
    export ${renv}=${v}
  fi
  if [ "${renv}" = "SDS_TASK_NAME" ]
  then
    export ${renv}="${!renv}-${1}"
  fi
done
if [ -z "${AP}" ]
then
  template=`cat fargate/sds-task.template.json`
else
  template=`cat fargate/sds-task.template-ap.json`
fi
envs=`env | grep SDS_`
envs="${envs} AWS_REGION"
for env in ${envs}
do
  OIFS=${IFS}
  IFS='='
  a=(${env})
  IFS=${OIFS}
  template="${template//\$\{${a}\}/${!a}}"
done
fn="fargate/sds-task.json.secret"
echo "${template}" | jq -e . > "${fn}" || exit 5
vim -c '%s/"${\(.*\)"/""/g' -c wq "${fn}"
cwd=`pwd`
if [ -z "${DRY}" ]
then
  aws ecs register-task-definition --cli-input-json "file://${cwd}/${fn}"
else
  cat "${cwd}/${fn}"
fi
