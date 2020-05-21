#!/bin/bash
if [ -z "${AWS_PROFILE}" ]
then
  echo "$0: you need to specify AWS_PROFILE=..."
  exit 1
fi
vpcid=`aws ec2 describe-vpcs | jq -r '.Vpcs[] | select(.CidrBlock == "10.0.0.0/16") | .VpcId'`
igwid=`aws ec2 describe-internet-gateways | jq -r ".InternetGateways[] | select(.Attachments[0].VpcId == \"${vpcid}\") | .InternetGatewayId"`
rtid=`aws ec2 describe-route-tables | jq -r ".RouteTables[] | select(.VpcId == \"${vpcid}\" and .VpcId == \"${vpcid}\" and (.Routes | length) == 2) | .RouteTableId"`
aws ec2 delete-route --destination-cidr-block 0.0.0.0/0 --route-table-id "${rtid}"
aws ec2 delete-route-table --route-table-id "${rtid}"
aws ec2 detach-internet-gateway --vpc-id "${vpcid}" --internet-gateway-id "${igwid}"
aws ec2 delete-internet-gateway --internet-gateway-id "${igwid}"
