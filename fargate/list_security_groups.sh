#!/bin/bash
if [ -z "${AWS_PROFILE}" ]
then
  echo "$0: you need to specify AWS_PROFILE=..."
  exit 1
fi
aws ec2 describe-security-groups | jq '.SecurityGroups[] | select(.Description == "SDS security group")'
