#!/bin/bash
# PUB=1 - use public AWS VPC address
if [ -z "${AWS_PROFILE}" ]
then
  echo "$0: you need to specify AWS_PROFILE=..."
  exit 1
fi
if [ -z "${1}" ]
then
  echo "$0: you need to specify env as a first argument: test|prod"
  exit 2
fi
if [ -z "${2}" ]
then
  echo "$0: you need to specify cluster name as a second argument"
  exit 3
fi
if [ -z "${3}" ]
then
  echo "$0: you need to specify service name as a 3rd argument"
  exit 4
fi
if [ -z "${4}" ]
then
  echo "$0: you need to specify task name as a 4th argument"
  exit 5
fi
for renv in SDS_VPC_ID SDS_SUBNET_ID
do
  if [ -z "${!renv}" ]
  then
    export ${renv}="`cat helm-charts/sds-helm/sds-helm/secrets/${renv}.secret`"
    if [ -z "${!renv}" ]
    then
      if [ "${renv}" = "SDS_VPC_ID" ]
      then
        export ${renv}=`aws ec2 describe-vpcs | jq -r '.Vpcs[] | select(.CidrBlock == "10.0.0.0/16") | .VpcId'`
      elif [ "${renv}" = "SDS_SUBNET_ID" ]
      then
        export ${renv}=`aws ec2 describe-subnets | jq -r '.Subnets[] | select(.CidrBlock == "10.0.128.0/17") | .SubnetId'`
      else
        echo "$0: you must specify ${renv}=... or provide helm-charts/sds-helm/sds-helm/secrets/${renv}.secret file (don't know how to get value from aws cli)"
        exit 6
      fi
    fi
    if [ -z "${!renv}" ]
    then
      echo "$0: you must specify ${renv}=... or provide helm-charts/sds-helm/sds-helm/secrets/${renv}.secret file (unable to get value from aws cli)"
      exit 7
    fi
  fi
done
echo "${SDS_VPC_ID}"
echo "${SDS_SUBNET_ID}"
exit 10
if [ -z "${PUB}" ]
then
  aws ecs create-service --cluster "${2}-${1}" --service-name "${3}-${1}" --task-definition "${4}-${1}:1" --desired-count 1 --launch-type "FARGATE" --network-configuration "awsvpcConfiguration={subnets=[subnet-${4}-${1}],securityGroups=[sg-${4}-${1}]}"
else
  aws ecs create-service --cluster "${2}-${1}" --service-name "${3}-${1}" --task-definition "${4}-${1}:1" --desired-count 1 --launch-type "FARGATE" --network-configuration "awsvpcConfiguration={subnets=[subnet-${4}-${1}],securityGroups=[sg-${4}-${1}],assignPublicIp=ENABLED}"
fi
