#!/bin/bash
# IDX=sds-foundation-project-datasource COL=id_in_repo ./scripts/compae_unique_values.sh
ES_URL="`cat helm-charts/sds-helm/sds-helm/secrets/ES_URL.prod.secret`" ./scripts/unique_values.sh > prod.txt || exit 1
ES_URL="`cat helm-charts/sds-helm/sds-helm/secrets/ES_URL.test.secret`" ./scripts/unique_values.sh > test.txt || exit 2
diff test.txt prod.txt > diff.txt
cat diff.txt
