#!/bin/bash
if [ -z "${AWS_PROFILE}" ]
then
  echo "$0: you need to specify AWS_PROFILE=..."
  exit 1
fi
lgname=`aws logs describe-log-groups | jq -r '.logGroups[] | select(.logGroupName == "sds-logs") .logGroupName'`
if [ ! -z "${lgname}" ]
then
  echo "Deleting log group ${lgname}"
  aws logs delete-log-group --log-group-name "${lgname}"
else
  echo 'No log group to delete'
fi
