#!/bin/bash
# NODES=4 - set number of nodes
# NS=sds - set namespace name, default sds
# FLAGS="esURL=\"`cat sds-helm/secrets/ES_URL_ext.secret`\",pvSize=30Gi"
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
if [ -z "$NS" ]
then
  NS=sds
fi
if [ -z "$FLAGS" ]
then
  FLAGS="foo=bar"
fi
change_namespace.sh $1 "$NS"
"${1}h.sh" install "${NS}-debug" ./sds-helm --set "namespace=$NS,deployEnv="$1",skipSecrets=1,skipCron=1,skipNamespace=1,skipPV=1,debugPod=1,stripErrorSize=4096,nodeNum=$NODES,nodeHash=$HSH,$FLAGS"
change_namespace.sh $1 default
