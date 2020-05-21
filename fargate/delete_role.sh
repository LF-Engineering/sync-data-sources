#!/bin/bash
if [ -z "${AWS_PROFILE}" ]
then
  echo "$0: you need to specify AWS_PROFILE=..."
  exit 1
fi
aws iam detach-role-policy --role-name ecsTaskExecutionRole --policy-arn arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy
aws iam delete-role --role-name ecsTaskExecutionRole
