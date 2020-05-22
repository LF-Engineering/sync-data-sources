#!/bin/bash
if [ -z "${AWS_PROFILE}" ]
then
  echo "$0: you need to specify AWS_PROFILE=..."
  exit 1
fi
igwid=`aws ec2 create-internet-gateway | jq -r '.InternetGateway.InternetGatewayId'`
if [ -z "${igwid}" ]
then
  echo 'Failed to create internet gateway'
  exit 2
fi
vpcid=`aws ec2 describe-vpcs | jq -r '.Vpcs[] | select(.CidrBlock == "10.0.0.0/16") | .VpcId'`
if [ -z "${vpcid}" ]
then
  echo 'Failed to find VPC'
  exit 3
fi
aws ec2 attach-internet-gateway --vpc-id "${vpcid}" --internet-gateway-id "${igwid}" || exit 4
rtid=`aws ec2 create-route-table --vpc-id "${vpcid}" | jq -r '.RouteTable.RouteTableId'`
if [ -z "${rtid}" ]
then
  echo 'Failed to create route table'
  exit 5
fi
aws ec2 create-route --route-table-id "${rtid}" --destination-cidr-block 0.0.0.0/0 --gateway-id "${igwid}" > /dev/null
#aws ec2 describe-route-tables --route-table-id "${rtid}"
