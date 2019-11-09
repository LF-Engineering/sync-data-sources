#!/bin/bash
if [ -z "${API_TOKEN}" ]
then
  echo -n "Slack legacy API token: "
  read -s API_TOKEN
fi
if [ -z "${PASS}" ]
then
  echo -n "SortingHat DB Password: "
  read -s PASS
fi
if [ -z "${IDENTITIES}" ]
then
  p2o.py --enrich --index slack_raw --index-enrich slack -e http://localhost:9200 --no_inc --debug --db-host localhost --db-sortinghat shdb --db-user shuser --db-password "${PASS}" slack C095YQBM2 --category message -t "${API_TOKEN}"
  p2o.py --enrich --index slack_raw --index-enrich slack -e http://localhost:9200 --no_inc --debug --db-host localhost --db-sortinghat shdb --db-user shuser --db-password "${PASS}" slack C095YQBLL --category message -t "${API_TOKEN}"
else
  echo "Only identities mode"
  p2o.py --only-enrich --refresh-identities --index slack_raw --index-enrich slack -e http://localhost:9200 --no_inc --debug --db-host localhost --db-sortinghat shdb --db-user shuser --db-password "${PASS}" slack C095YQBM2 --category message -t "${API_TOKEN}"
  p2o.py --only-enrich --refresh-identities --index slack_raw --index-enrich slack -e http://localhost:9200 --no_inc --debug --db-host localhost --db-sortinghat shdb --db-user shuser --db-password "${PASS}" slack C095YQBLL --category message -t "${API_TOKEN}"
fi
