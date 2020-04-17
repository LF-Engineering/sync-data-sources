#!/bin/bash
if [ -z "$1" ]
then
  echo 'Please specify elasticsearch url as a first argument'
  exit 1
fi
if [ -z "$2" ]
then
  echo 'Please specify max_compilations_rate as a second argument (example: 100/1m)'
  exit 2
fi
err=err.txt
function cleanup {
  rm -f "$err"
}
trap cleanup EXIT
function fexit {
  cat $err
  echo "$1"
  exit $2
}
json="{\"persistent\":{\"script.max_compilations_rate\":\"${2}\"},\"transient\":{\"script.max_compilations_rate\":\"${2}\"}}"
curl -XPUT -H 'Content-Type: application/json' "${1}/_cluster/settings" -d "$json" 2>"$err" || fexit "Error setting max compilations setting: $json" 2
curl "${1}/_cluster/settings?pretty" 2>"$err" | grep max_compilations
