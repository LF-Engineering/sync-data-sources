#!/bin/bash
if [ -z "${PASS}" ]
then
  echo -n "SortingHat DB Password: "
  read -s PASS
fi
if [ -z "${IDENTITIES}" ]
then
  p2o.py --enrich --index confluence_raw --index-enrich confluence -e http://localhost:9200 --no_inc --debug --db-host localhost --db-sortinghat shdb --db-user shuser --db-password "${PASS}" confluence 'http://wiki.onap.org' --category 'historical content'
else
  echo "Only identities mode"
  p2o.py --only-enrich --refresh-identities --index confluence_raw --index-enrich confluence -e http://localhost:9200 --no_inc --debug --db-host localhost --db-sortinghat shdb --db-user shuser --db-password "${PASS}" confluence 'http://wiki.onap.org' --category 'historical content'
fi
