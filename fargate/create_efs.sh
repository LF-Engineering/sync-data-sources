#!/bin/bash
# AP=1 create access point too
# SS=1 set secret mode
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
if [ -z "${fsid}" ]
then
  echo 'Failed to create file system'
  exit 3
fi
echo "FileSystemId: ${fsid}"
if [ ! -z "${AP}" ]
then
  apid=`aws efs create-access-point --client-token sds-efs-access-point --tags "Key=Name,Value=sds-efs-access-point" --file-system-id "${fsid}" | jq -r '.AccessPointId'`
  if [ -z "${apid}" ]
  then
    echo 'Failed to create access point'
    exit 4
  fi
  echo "AccessPointId: ${apid}"
fi
subnetid=`aws ec2 describe-subnets | jq -r '.Subnets[] | select(.CidrBlock == "10.0.128.0/17") | .SubnetId'`
if [ -z "${subnetid}" ]
then
  echo 'Failed to find subnet'
  exit 5
fi
sgidmt=`aws ec2 describe-security-groups | jq -r '.SecurityGroups[] | select(.Description == "SDS EFS MT security group") | .GroupId'`
if [ -z "${sgidmt}" ]
then
  echo 'Failed to find EFS MT security group'
  exit 6
fi
aws efs create-mount-target --file-system-id "${fsid}" --subnet-id "${subnetid}" --security-group "${sgidmt}" --region "${AWS_REGION}" || exit 7
if [ ! -z "${SS}" ]
then
  echo -n "${fsid}" > "helm-charts/sds-helm/sds-helm/secrets/SDS_FS_ID.secret"
  if [ ! -z "${AP}" ]
  then
    echo -n "${apid}" > "helm-charts/sds-helm/sds-helm/secrets/SDS_FSAP_ID.secret"
  fi
fi
