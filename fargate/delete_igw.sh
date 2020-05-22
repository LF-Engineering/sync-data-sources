#!/bin/bash
if [ -z "${AWS_PROFILE}" ]
then
  echo "$0: you need to specify AWS_PROFILE=..."
  exit 1
fi
vpcid=`aws ec2 describe-vpcs | jq -r '.Vpcs[] | select(.CidrBlock == "10.0.0.0/16") | .VpcId'`
if [ -z "${vpcid}" ]
then
  echo 'No VPC found, not deleting internet gateway and route/route tables'
  exit 0
fi
igwid=`aws ec2 describe-internet-gateways | jq -r ".InternetGateways[] | select(.Attachments[0].VpcId == \"${vpcid}\") | .InternetGatewayId"`
rtid=`aws ec2 describe-route-tables | jq -r ".RouteTables[] | select(.VpcId == \"${vpcid}\" and (.Routes | length) == 2) | .RouteTableId"`
if [ ! -z "${rtid}" ]
then
  echo "Deleting route and routetable ${rtid}"
  aws ec2 delete-route --destination-cidr-block 0.0.0.0/0 --route-table-id "${rtid}"
  aws ec2 delete-route-table --route-table-id "${rtid}"
else
  echo 'No route and route table to delete'
fi
if [ ! -z "${igwid}" ]
then
  echo "Detaching & deleting internet gateway ${igwid}"
  aws ec2 detach-internet-gateway --vpc-id "${vpcid}" --internet-gateway-id "${igwid}"
  aws ec2 delete-internet-gateway --internet-gateway-id "${igwid}"
else
  echo 'No internet gateway to detach & delete'
fi
