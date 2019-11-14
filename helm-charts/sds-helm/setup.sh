#!/bin/bash
# DBG=1 deploy with debug pod installed
# ES_BULK_SIZE=10000 - set ES bulk size
# NODES=4 - set number of nodes
# DRY=1 - dry run mode
if [ -z "$1" ]
then
  echo "$0: you need to specify env: test, dev, stg, prod"
  exit 1
fi
if [ -z "$NODES" ]
then
  export NODES=1
  export HSH=''
else
  export HSH='1'
fi
if [ -z "$DRY" ]
then
  "${1}h.sh" install sds-namespace ./sds-helm --set "skipSecrets=1,skipCron=1,skipPV=1,nodeNum=$NODES,nodeHash=$HSH"
  change_namespace.sh $1 sds
  "${1}h.sh" install sds ./sds-helm --set "deployEnv=$1,skipNamespace=1,debugPod=$DBG,esBulkSize=$ES_BULK_SIZE,nodeNum=$NODES,nodeHash=$HSH"
  change_namespace.sh $1 default
else
  "${1}h.sh" install --debug --dry-run --generate-name ./sds-helm --set "deployEnv=$1,debugPod=$DBG,esBulkSize=$ES_BULK_SIZE,nodeNum=$NODES,nodeHash=$HSH"
fi
