#!/bin/bash
NS=sds-ext ./debug_delete.sh prod
NS=sds-ext ./debug_delete.sh test
NS=sds ./debug_delete.sh prod
NS=sds ./debug_delete.sh test
sleep 60
prodk.sh -n sds-ext get po
testk.sh -n sds-ext get po
prodk.sh -n sds get po
testk.sh -n sds get po
NS=sds-ext NODES=2 FLAGS=esURL=`cat sds-helm/secrets/ES_URL_Fayaz_prod.secret` ./debug.sh prod
NS=sds-ext NODES=2 FLAGS=esURL=`cat sds-helm/secrets/ES_URL_Fayaz_test.secret` ./debug.sh test
NS=sds NODES=2 ./debug.sh prod
NS=sds NODES=2 ./debug.sh test
