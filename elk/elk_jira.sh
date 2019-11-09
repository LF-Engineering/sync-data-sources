#!/bin/bash
if [ -z "${JIRA_USER}" ]
then
  echo -n "Jira user: "
  read JIRA_USER
fi
if [ -z "${JIRA_PWD}" ]
then
  echo -n "Jira Password: "
  read -s JIRA_PWD
fi
if [ -z "${PASS}" ]
then
  echo -n "SortingHat DB Password: "
  read -s PASS
fi
if [ -z "${IDENTITIES}" ]
then
  p2o.py --enrich --index jira_raw --index-enrich jira -e http://localhost:9200 --no_inc --debug --db-host localhost --db-sortinghat shdb --db-user shuser --db-password "${PASS}" jira 'http://jira.onosproject.org' --category issue --backend-user "${JIRA_USER}" --backend-password "${JIRA_PWD}" --verify False
else
  echo "Only identities mode"
  p2o.py --only-enrich --refresh-identities --index jira_raw --index-enrich jira -e http://localhost:9200 --no_inc --debug --db-host localhost --db-sortinghat shdb --db-user shuser --db-password "${PASS}" jira 'http://jira.onosproject.org' --category issue --backend-user "${JIRA_USER}" --backend-password "${JIRA_PWD}" --verify False
fi
