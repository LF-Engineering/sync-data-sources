#!/bin/bash
if [ -z "${AWS_PROFILE}" ]
then
  echo "$0: you need to specify AWS_PROFILE=..."
  exit 1
fi
vpcid=`aws ec2 describe-vpcs | jq -r '.Vpcs[] | select(.CidrBlock == "10.0.0.0/16") | .VpcId'`
aws ec2 describe-internet-gateways | jq ".InternetGateways[] | select(.Attachments[0].VpcId == \"${vpcid}\")"
aws ec2 describe-route-tables | jq ".RouteTables[] | select(.VpcId == \"${vpcid}\" and .VpcId == \"${vpcid}\" and (.Routes | length) == 2)"
