#!/bin/bash
if [ -z "${AWS_PROFILE}" ]
then
  echo "$0: you need to specify AWS_PROFILE=..."
  exit 1
fi
roleid=`aws iam list-roles | jq -r '.Roles[] | select(.RoleName == "ecsTaskExecutionRole") | .RoleId'`
if [ ! -z "${roleid}" ]
then
  aws iam list-roles | jq '.Roles[] | select(.RoleName == "ecsTaskExecutionRole")'
  aws iam list-attached-role-policies --role-name ecsTaskExecutionRole | jq .
fi
