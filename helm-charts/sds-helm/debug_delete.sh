#!/bin/bash
if [ -z "$1" ]
then
  echo "$0: you need to specify env: test, dev, stg, prod"
  exit 1
fi
change_namespace.sh $1 sds
"${1}h.sh" delete sds-debug
change_namespace.sh $1 default
