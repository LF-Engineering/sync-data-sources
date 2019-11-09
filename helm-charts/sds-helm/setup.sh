#!/bin/bash
if  [ -z "$1" ]
then
  echo "$0: you need to specify env: test, dev, stg, prod"
  exit 1
fi
"${1}h.sh" install sds-namespace ./sds-helm --set "skipSecrets=1,skipCron=1,skipPV=1"
change_namespace.sh $1 sds
"${1}h.sh" install sds ./sds-helm --set "deployEnv=$1,skipNamespace=1"
change_namespace.sh $1 default
