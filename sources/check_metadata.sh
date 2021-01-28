#!/bin/bash
ES="`cat ../helm-charts/sds-helm/sds-helm/secrets/ES_URL.prod.secret`"
for f in `cat md_indices.secret`
do
  if [[ ${f} == *"jira"* ]]
  then
    echo "jira index: $f"
    curl -s -XPOST -H 'Content-type: application/json' "${ES}/_sql?format=csv" -d"{\"query\":\"select workinggroup, count(*) as cnt from \\\"${f}\\\" group by workinggroup order by workinggroup\"}"
  elif [[ ${f} == *"finosmeetings"* ]]
  then
    echo "finosmeetings index: $f"
    curl -s -XPOST -H 'Content-type: application/json' "${ES}/_sql?format=csv" -d"{\"query\":\"select workinggroup, count(*) as cnt from \\\"${f}\\\" group by workinggroup order by workinggroup\"}"
  else
    echo "index: $f"
    curl -s -XPOST -H 'Content-type: application/json' "${ES}/_sql?format=txt" -d"{\"query\":\"select workinggroup, meta_title, meta_type, meta_program, meta_contributed, meta_state, count(*) as cnt from \\\"${f}\\\" group by workinggroup, meta_title, meta_type, meta_program, meta_contributed, meta_state order by workinggroup, meta_title, meta_type, meta_program, meta_contributed, meta_state\"}"
  fi
  echo "======================="
done
