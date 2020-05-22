#!/bin/bash
# SS=1 set secret mode
if [ -z "${AWS_PROFILE}" ]
then
  echo "$0: you need to specify AWS_PROFILE=..."
  exit 1
fi
if [ -z "${AWS_REGION}" ]
then
  echo "$0: you need to specify AWS_REGION=..."
  exit 1
fi
vpcid=`aws ec2 describe-vpcs | jq -r '.Vpcs[] | select(.CidrBlock == "10.0.0.0/16") | .VpcId'`
if [ -z "${vpcid}" ]
then
  echo 'Failed to find VPC'
  exit 2
fi
sgid=`aws ec2 create-security-group --group-name sds-sg --description "SDS security group" --vpc-id "${vpcid}" | jq -r '.GroupId'`
if [ -z "${sgid}" ]
then
  echo 'Failed to create main security group'
  exit 3
fi
sgidmt=`aws ec2 create-security-group --group-name sds-sg-mt --description "SDS EFS MT security group" --vpc-id "${vpcid}" | jq -r '.GroupId'`
if [ -z "${sgidmt}" ]
then
  echo 'Failed to create EFS MT security group'
  exit 4
fi
aws ec2 authorize-security-group-ingress --group-id "${sgid}" --protocol tcp --port 22 --cidr 0.0.0.0/0 --region "${AWS_REGION}" || exit 5
aws ec2 authorize-security-group-ingress --group-id "${sgidmt}" --protocol tcp --port 2049 --source-group "${sgid}" --region "${AWS_REGION}" || exit 6
if [ ! -z "${SS}" ]
then
  echo -n "${sgid}" > "helm-charts/sds-helm/sds-helm/secrets/SDS_SG_ID.secret"
  echo -n "${sgidmt}" > "helm-charts/sds-helm/sds-helm/secrets/SDS_SGMT_ID.secret"
fi
