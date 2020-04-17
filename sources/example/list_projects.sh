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
curl -H 'Content-Type: application/json' "${ES_URL}/${1}/_doc/_search?size=0" -d '{"aggs":{"project":{"terms":{"field":"project","size":2147483647}}}}' 2>/dev/null | jq '.aggregations.project.buckets'
