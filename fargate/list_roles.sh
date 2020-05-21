#!/bin/bash
if [ -z "${AWS_PROFILE}" ]
then
  echo "$0: you need to specify AWS_PROFILE=..."
  exit 1
fi
aws iam list-roles | jq '.Roles[] | select(.RoleName == "ecsTaskExecutionRole")'
aws iam list-attached-role-policies --role-name ecsTaskExecutionRole
