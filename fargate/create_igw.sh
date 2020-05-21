#!/bin/bash
if [ -z "${AWS_PROFILE}" ]
then
  echo "$0: you need to specify AWS_PROFILE=..."
  exit 1
fi
igwid=`aws ec2 create-internet-gateway | jq -r '.InternetGateway.InternetGatewayId'`
vpcid=`aws ec2 describe-vpcs | jq -r '.Vpcs[] | select(.CidrBlock == "10.0.0.0/16") | .VpcId'`
aws ec2 attach-internet-gateway --vpc-id "${vpcid}" --internet-gateway-id "${igwid}"
rtid=`aws ec2 create-route-table --vpc-id "${vpcid}" | jq -r '.RouteTable.RouteTableId'`
aws ec2 create-route --route-table-id "${rtid}" --destination-cidr-block 0.0.0.0/0 --gateway-id "${igwid}"
aws ec2 describe-route-tables --route-table-id "${rtid}"
