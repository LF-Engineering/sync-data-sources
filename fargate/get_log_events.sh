#!/bin/bash
if [ -z "${AWS_PROFILE}" ]
then
  echo "$0: you need to specify AWS_PROFILE=..."
  exit 1
fi
if [ -z "${1}" ]
then
  echo "$0: you need to specify log group name as a first argument"
  exit 2
fi
if [ -z "${2}" ]
then
  echo "$0: you need to specify log stream name as a second argument"
  exit 3
fi
aws logs get-log-events --log-group-name "${1}" --log-stream-name "${2}" | jq -r '.events[] | .message'
