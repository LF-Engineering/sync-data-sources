#!/bin/bash
if [ -z "$1" ]
then
  echo "$0: you need to provide 1st argument: date-from in YYYY-MM-DD format"
  exit 1
fi
if [ -z "$2" ]
then
  echo "$0: you need to provide 2nd argument: date-to in YYYY-MM-DD format"
  exit 2
fi
if [ -z "$3" ]
then
  echo "$0: you need to provide 3rd argument: org/repo"
  exit 3
fi
ary=(${3//\// })
org=${ary[0]}
repo="${org}/${ary[1]}"
function finish {
  cat /tmp/bq.sql
  rm -f /tmp/bq.sql
}
trap finish EXIT
cp BigQuery/prs.sql /tmp/bq.sql || exit 4
FROM="{{dtfrom}}" TO="$1" MODE=ss replacer /tmp/bq.sql || exit 5
FROM="{{dtto}}" TO="$2" MODE=ss replacer /tmp/bq.sql || exit 6
FROM="{{org}}" TO="$org" MODE=ss replacer /tmp/bq.sql || exit 7
FROM="{{repo}}" TO="$repo" MODE=ss replacer /tmp/bq.sql || exit 8
ofn="prs_${1//-/}_${2//-/}.csv"
echo "$ofn"
cat /tmp/bq.sql | bq --format=csv --headless query --use_legacy_sql=true -n 1000000 --use_cache > "$ofn" || exit 9
#ed "$ofn" <<<$'1d\nwq\n' || exit 8
echo "$ofn written"
