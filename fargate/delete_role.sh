#!/bin/bash
if [ -z "${AWS_PROFILE}" ]
then
  echo "$0: you need to specify AWS_PROFILE=..."
  exit 1
fi
roleid=`aws iam list-roles | jq -r '.Roles[] | select(.RoleName == "ecsTaskExecutionRole") | .RoleId'`
if [ ! -z "${roleid}" ]
then
  echo "Deleting role and role policy ${roleid}"
  aws iam detach-role-policy --role-name ecsTaskExecutionRole --policy-arn arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy
  aws iam delete-role --role-name ecsTaskExecutionRole
else
  echo 'No role and role policy to delete'
fi
