#!/bin/bash
if [ -z "${AWS_PROFILE}" ]
then
  echo "${0}: you need to specify AWS_PROFILE=..."
  exit 1
fi
cwd=`pwd`
aws iam create-role --role-name ecsTaskExecutionRole --assume-role-policy-document "file://${cwd}/fargate/ecsTaskExecutionRole.json"
aws iam attach-role-policy --role-name ecsTaskExecutionRole --policy-arn arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy
