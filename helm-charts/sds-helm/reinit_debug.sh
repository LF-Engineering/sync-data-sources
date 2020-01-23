#!/bin/bash
if [ -z "$1" ]
then
  echo "$0: you need to specify wait time as an arg, for example 120"
  exit 1
fi
./debug_delete.sh prod
./debug_delete.sh test
#NS=sds ./debug_delete.sh prod
#NS=sds ./debug_delete.sh test
sleep 60
prodk.sh -n sds get po
testk.sh -n sds get po
#prodk.sh -n sds get po
#testk.sh -n sds get po
#NODES=7 FLAGS=esURL=`cat sds-helm/secrets/ES_URL_Fayaz_prod.secret` ./debug.sh prod
NODES=7 ./debug.sh prod
NODES=7 ./debug.sh test
#NS=sds NODES=7 ./debug.sh prod
#NS=sds NODES=7 ./debug.sh test
