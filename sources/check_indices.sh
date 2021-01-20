#!/bin/bash
clear; curl -s http://127.0.0.1:19200/_cat/aliases | sort > ~/idx; curl -s http://127.0.0.1:19200/_cat/indices | sort >> ~/idx; cat ~/idx
#for f in `cat idx`; do echo "index $f"; curl -s "http://127.0.0.1:19200/$f/_search?size=100000" | jq '.hits.hits[]._source.github_repo' | sort | uniq; done > out
for f in `cat ~/idx`; do echo "index $f"; curl -s "http://127.0.0.1:19200/$f/_search?size=100000" | jq '.hits.hits[]._source.origin' | sort | uniq; done > ~/report.txt
