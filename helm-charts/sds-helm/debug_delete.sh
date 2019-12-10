#!/bin/bash
# NS=sds - set namespace name, default sds
if [ -z "$1" ]
then
  echo "$0: you need to specify env: test, dev, stg, prod"
  exit 1
fi
if [ -z "$NS" ]
then
  NS=sds
fi
change_namespace.sh $1 "$NS"
"${1}h.sh" delete "${NS}-debug"
change_namespace.sh $1 default
