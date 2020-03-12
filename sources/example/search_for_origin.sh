#!/bin/bash
if [ -z "$ES_URL" ]
then
  echo "$0: you must set ES_URL=... to run this script"
  exit 1
fi
if [ -z "$1" ]
then
  echo "$0: please specify index name as a 1st arg"
  exit 2
fi
if [ -z "$2" ]
then
  echo "$0: please specify origin as a 2nd arg"
  exit 3
fi
curl -H 'Content-Type: application/json' "${ES_URL}/${1}/_doc/_search?size=1000" -d "{\"query\":{\"query_string\":{\"query\":\"origin:\\\"${2}\\\"\"}}}" 2>/dev/null | jq
