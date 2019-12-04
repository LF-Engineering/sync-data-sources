#!/bin/bash
if [ -z "$1" ]
then
  echo "$0: you need to specify env: test, dev, stg, prod"
  exit 1
fi
change_namespace.sh $1 sds
"${1}h.sh" install sds-debug ./sds-helm --set deployEnv="$1",skipSecrets=1,skipCron=1,skipNamespace=1,skipPV=1,debugPod=1,stripErrorSize=4096
change_namespace.sh $1 default
