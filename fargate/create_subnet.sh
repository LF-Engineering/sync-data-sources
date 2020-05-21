#!/bin/bash
if [ -z "${AWS_PROFILE}" ]
then
  echo "$0: you need to specify AWS_PROFILE=..."
  exit 1
fi
vpcid=`aws ec2 describe-vpcs | jq -r '.Vpcs[] | select(.CidrBlock == "10.0.0.0/16") | .VpcId'`
subnetid=`aws ec2 create-subnet --vpc-id "${vpcid}" --cidr-block 10.0.128.0/17 | jq -r '.Subnet.SubnetId'`
rtid=`aws ec2 describe-route-tables | jq -r ".RouteTables[] | select(.VpcId == \"${vpcid}\" and .VpcId == \"${vpcid}\" and (.Routes | length) == 2) | .RouteTableId"`
echo "SubnetId: $subnetid"
aws ec2 associate-route-table  --subnet-id "${subnetid}" --route-table-id "${rtid}"
aws ec2 modify-subnet-attribute --subnet-id "${subnetid}" --map-public-ip-on-launch
