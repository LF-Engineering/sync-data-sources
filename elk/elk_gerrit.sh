#!/bin/bash
if [ -z "${PASS}" ]
then
  echo -n "SortingHat DB Password: "
  read -s PASS
fi
if [ -z "${GERRIT_USER}" ]
then
  echo -n "Gerrit Password: "
  read GERRIT_USER
fi
if [ -z "${IDENTITIES}" ]
then
  p2o.py --enrich --index gerrit_raw --index-enrich gerrit -e http://localhost:9200 --no_inc --debug --db-host localhost --db-sortinghat shdb --db-user shuser --db-password "${PASS}" gerrit --user "${GERRIT_USER}" --category review gerrit.onosproject.org
else
  echo "Only identities mode"
  p2o.py --only-enrich --refresh-identities --index gerrit_raw --index-enrich gerrit -e http://localhost:9200 --no_inc --debug --db-host localhost --db-sortinghat shdb --db-user shuser --db-password "${PASS}" gerrit --user "${GERRIT_USER}" --category review gerrit.onosproject.org
fi
