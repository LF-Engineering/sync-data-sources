#!/bin/bash
if [ -z "${AWS_PROFILE}" ]
then
  echo "$0: you need to specify AWS_PROFILE=..."
  exit 1
fi
vpcid=`aws ec2 describe-vpcs | jq -r '.Vpcs[] | select(.CidrBlock == "10.0.0.0/16") | .VpcId'`
if [ ! -z "${vpcid}" ]
then
  echo "Deleting vpc ${vpcid}"
  aws ec2 delete-vpc --vpc-id "${vpcid}"
else
  echo 'No VPC to delete'
fi
