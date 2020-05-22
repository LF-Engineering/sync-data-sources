#!/bin/bash
# SS=1 set secret mode
if [ -z "${AWS_PROFILE}" ]
then
  echo "${0}: you need to specify AWS_PROFILE=..."
  exit 1
fi
cwd=`pwd`
arn=`aws iam create-role --role-name ecsTaskExecutionRole --assume-role-policy-document "file://${cwd}/fargate/ecsTaskExecutionRole.json" | jq -r '.Role.Arn'`
echo "Role Arn: ${arn}"
if [ ! -z "${arn}" ]
then
  aws iam attach-role-policy --role-name ecsTaskExecutionRole --policy-arn 'arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy' || exit 3
  if [ ! -z "${SS}" ]
  then
    echo -n "${arn}" > "helm-charts/sds-helm/sds-helm/secrets/SDS_ROLE_ARN.secret"
  fi
else
  echo 'Failed to create role'
  exit 2
fi
