#!/bin/bash
if [ -z "${ES_URL}" ]
then
  echo "$0: you need to set ES_URL=..."
  exit 1
fi
if [ -z "$1" ]
then
  echo "$0: you need to specify some text to search for as an 1st argument"
  exit 2
fi
if [ -z "$LIMIT" ]
then
  LIMIT=20
fi
echo curl -s -XPOST -H 'Content-Type: application/json' "${ES_URL}/_sql?format=csv" -d"{\"query\":\"select * from \\\"sdslog\\\" where msg like '%${1}%' limit ${LIMIT}\"}"
curl -s -XPOST -H 'Content-Type: application/json' "${ES_URL}/_sql?format=csv" -d"{\"query\":\"select * from \\\"sdslog\\\" where msg like '%${1}%' limit ${LIMIT}\"}"
