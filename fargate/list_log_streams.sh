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
aws logs describe-log-streams --log-group-name "${1}" 2>/dev/null | jq '.logStreams[] | .logStreamName'
