#!/bin/bash
if [ -z "$ES_URL" ]
then
  echo "$0: please specify ES_URL=..."
  exit 1
fi
curl "${ES_URL}/sdsmtx/_search?pretty"
curl -XPOST -H 'Content-Type: application/json' "${ES_URL}/sdsmtx/_delete_by_query?conflicts=proceed&refresh=true" -d'{"query":{"query_string":{"query":"mtx:\"rename-node-1\""}}}'
curl -XPOST -H 'Content-Type: application/json' "${ES_URL}/sdsmtx/_delete_by_query?conflicts=proceed&refresh=true" -d'{"query":{"query_string":{"query":"mtx:\"rename-node-2\""}}}'
curl -XPOST -H 'Content-Type: application/json' "${ES_URL}/sdsmtx/_delete_by_query?conflicts=proceed&refresh=true" -d'{"query":{"query_string":{"query":"mtx:\"rename-node-3\""}}}'
curl "${ES_URL}/sdsmtx/_search?pretty"
