#!/bin/bash
# ES_URL=... IDX=sds-foundation-project-datasource ORIGIN=https://github.com/org/repo COL=id_in_repo ./unique_values.sh
fn="`echo $RANDOM`.json"
function cleanup {
  #cat $fn
  rm -f $fn
}
trap cleanup EXIT
echo -n "{\"query\":\"select ${COL} from \\\"${IDX}\\\" where origin = '${ORIGIN}' group by ${COL} order by ${COL}\"}" > $fn
data=`curl -s -XPOST -H 'Content-type: application/json' "${ES_URL}/_sql?format=json" -d@$fn`
cursor=`echo $data | jq -r .cursor`
vals=''
n=0
while [ ! "$cursor" = "null" ]
do
  for v in `echo $data | jq -r .rows[][0]`
  do
    if [ -z "$vals" ]
    then
      vals=$v
    else
      vals="${vals} ${v}"
    fi
    n=$((n+1))
  done
  echo -n "{\"cursor\":\"$cursor\"}" > $fn
  data=`curl -s -XPOST -H 'Content-type: application/json' "${ES_URL}/_sql?format=json" -d@$fn`
  cursor=`echo $data | jq -r .cursor`
done
echo "${n} values:"
svals=$(echo "$vals" | tr " " "\n" | sort -n | uniq | tr "\n" " ")
for v in $svals
do
  echo $v
done
