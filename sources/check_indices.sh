#!/bin/bash
clear; curl -s http://127.0.0.1:19200/_cat/aliases | sort; curl -s http://127.0.0.1:19200/_cat/indices | sort
for f in `cat idx`; do echo "index $f"; curl -s "http://127.0.0.1:19200/$f/_search?size=10000" | jq '.hits.hits[]._source.github_repo' | sort | uniq; done > out
