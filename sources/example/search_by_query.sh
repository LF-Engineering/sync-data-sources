#!/bin/bash
curl -H 'Content-Type: application/json' "${ES_URL}/bitergia-git-aoc_onap_enriched_191112/_doc/_search?size=1000" -d '{"query":{"query_string":{"query":"origin:(\"https://gerrit.onap.org/r/vnfsdk/refrepo\" OR \"https://gerrit.onap.org/r/ooma\" OR \"https://gerrit.onap.org/r/doca\")"}}}' 2>/dev/null | jq
