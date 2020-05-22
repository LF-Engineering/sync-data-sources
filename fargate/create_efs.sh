#!/bin/bash
# AP=1 - create access point too
if [ -z "${AWS_PROFILE}" ]
then
  echo "$0: you need to specify AWS_PROFILE=..."
  exit 1
fi
if [ -z "${AWS_REGION}" ]
then
  echo "$0: you need to specify AWS_REGION=..."
  exit 2
fi
fsid=`aws efs create-file-system --region "${AWS_REGION}" --creation-token sds-efs-volume --performance-mode maxIO --tags "Key=Name,Value=sds-efs-volume" | jq -r '.FileSystemId'`
echo "FileSystemId: ${fsid}"
if [ ! -z "${AP}" ]
then
  aws efs create-access-point --client-token sds-efs-access-point --tags "Key=Name,Value=sds-efs-access-point" --file-system-id "${fsid}"
fi
subnetid=`aws ec2 describe-subnets | jq -r '.Subnets[] | select(.CidrBlock == "10.0.128.0/17") | .SubnetId'`
sgidmt=`aws ec2 describe-security-groups | jq -r '.SecurityGroups[] | select(.Description == "SDS EFS MT security group") | .GroupId'`
aws efs create-mount-target --file-system-id "${fsid}" --subnet-id "${subnetid}" --security-group "${sgidmt}" --region "${AWS_REGION}"
