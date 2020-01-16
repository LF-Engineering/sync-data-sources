NS=sds-ext ./debug_delete.sh prod
NS=sds-ext ./debug_delete.sh test
sleep 30
NS=sds-ext NODES=2 FLAGS=esURL=`cat sds-helm/secrets/ES_URL_Fayaz_prod.secret` ./debug.sh prod
NS=sds-ext NODES=2 FLAGS=esURL=`cat sds-helm/secrets/ES_URL_Fayaz_test.secret` ./debug.sh test
