#!/bin/bash
# SS=1 set secret mode
if [ -z "${AWS_PROFILE}" ]
then
  echo "$0: you need to specify AWS_PROFILE=..."
  exit 1
fi
vpcid=`aws ec2 describe-vpcs | jq -r '.Vpcs[] | select(.CidrBlock == "10.0.0.0/16") | .VpcId'`
if [ -z "${vpcid}" ]
then
  echo 'Failed to find VPC'
  exit 2
fi
subnetid=`aws ec2 create-subnet --vpc-id "${vpcid}" --cidr-block 10.0.128.0/17 | jq -r '.Subnet.SubnetId'`
if [ -z "${subnetid}" ]
then
  echo 'Failed to create a subnet'
  exit 3
fi
rtid=`aws ec2 describe-route-tables | jq -r ".RouteTables[] | select(.VpcId == \"${vpcid}\" and .VpcId == \"${vpcid}\" and (.Routes | length) == 2) | .RouteTableId"`
if [ -z "${rtid}" ]
then
  echo 'Failed to find route table'
  exit 4
fi
echo "SubnetId: $subnetid"
aws ec2 associate-route-table  --subnet-id "${subnetid}" --route-table-id "${rtid}" || exit 5
aws ec2 modify-subnet-attribute --subnet-id "${subnetid}" --map-public-ip-on-launch || exit 6
if [ ! -z "${SS}" ]
then
  echo -n "${subnetid}" > "helm-charts/sds-helm/sds-helm/secrets/SDS_SUBNET_ID.secret"
fi
