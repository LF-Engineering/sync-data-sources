#!/bin/bash
if [ -z "${AWS_PROFILE}" ]
then
  echo "$0: you need to specify AWS_PROFILE=..."
  exit 1
fi
subnetid=`aws ec2 describe-subnets | jq -r '.Subnets[] | select(.CidrBlock == "10.0.128.0/17") | .SubnetId'`
if [ ! -z "${subnetid}" ]
then
  echo "Deleting subnet ${subnetid}"
  aws ec2 delete-subnet --subnet-id "${subnetid}"
else
  echo 'No subnet to delete'
fi
