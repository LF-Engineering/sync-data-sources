#!/bin/bash
if [ -z "${AWS_PROFILE}" ]
then
  echo "$0: you need to specify AWS_PROFILE=..."
  exit 1
fi
sgid=`aws ec2 describe-security-groups | jq -r '.SecurityGroups[] | select(.Description == "SDS security group") | .GroupId'`
aws ec2 delete-security-group --group-id "${sgid}"
