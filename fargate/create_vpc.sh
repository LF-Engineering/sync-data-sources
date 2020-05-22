#!/bin/bash
# SS=1 set secret mode
if [ -z "${AWS_PROFILE}" ]
then
  echo "$0: you need to specify AWS_PROFILE=..."
  exit 1
fi
vpcid=`aws ec2 create-vpc --cidr-block 10.0.0.0/16 | jq -r '.Vpc.VpcId'`
if [ ! -z "${vpcid}" ]
then
  echo "VPC ${vpcid}"
  aws ec2 modify-vpc-attribute --vpc-id "${vpcid}" --enable-dns-hostnames || exit 2
  aws ec2 modify-vpc-attribute --vpc-id "${vpcid}" --enable-dns-support || exit 3
  if [ ! -z "${SS}" ]
  then
    echo -n "${vpcid}" > "helm-charts/sds-helm/sds-helm/secrets/SDS_VPC_ID.secret"
  fi
else
  echo 'Failed to create VPC'
  exit 4
fi
