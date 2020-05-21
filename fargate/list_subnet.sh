#!/bin/bash
if [ -z "${AWS_PROFILE}" ]
then
  echo "$0: you need to specify AWS_PROFILE=..."
  exit 1
fi
aws ec2 describe-subnets | jq '.Subnets[] | select(.CidrBlock == "10.0.128.0/17")'
