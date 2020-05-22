#!/bin/bash
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
sgid=`aws ec2 create-security-group --group-name sds-sg --description "SDS security group" --vpc-id "${vpcid}" | jq -r '.GroupId'`
sgidmt=`aws ec2 create-security-group --group-name sds-sg-mt --description "SDS EFS MT security group" --vpc-id "${vpcid}" | jq -r '.GroupId'`
aws ec2 authorize-security-group-ingress --group-id "${sgid}" --protocol tcp --port 22 --cidr 0.0.0.0/0 --region "${AWS_REGION}"
aws ec2 authorize-security-group-ingress --group-id "${sgidmt}" --protocol tcp --port 2049 --source-group "${sgid}" --region "${AWS_REGION}"
