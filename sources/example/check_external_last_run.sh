#!/bin/bash
if [ -z "$ES_URL" ]
then
  echo "$0: you must set ES_URL=... to run this script"
  exit 1
fi
curl -H 'Content-Type: application/json' "${ES_URL}/sdsdata/_doc/_search?size=10" -d '{"query":{"query_string":{"query":"index:\"bitergia\" AND endpoint:\"external\" AND type:\"last_sync\""}}}' 2>/dev/null | jq
