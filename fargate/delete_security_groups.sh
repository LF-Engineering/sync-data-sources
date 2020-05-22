#!/bin/bash
if [ -z "${AWS_PROFILE}" ]
then
  echo "$0: you need to specify AWS_PROFILE=..."
  exit 1
fi
sgidmt=`aws ec2 describe-security-groups | jq -r '.SecurityGroups[] | select(.Description == "SDS EFS MT security group") | .GroupId'`
sgid=`aws ec2 describe-security-groups | jq -r '.SecurityGroups[] | select(.Description == "SDS security group") | .GroupId'`
if [ ! -z "${sgidmt}" ]
then
  echo "Deleting EFS MT security group ${sgidmt}"
  aws ec2 delete-security-group --group-id "${sgidmt}"
else
  echo 'No EFS MT security group to delete'
fi
if [ ! -z "${sgid}" ]
then
  echo "Deleting main security group ${sgid}"
  aws ec2 delete-security-group --group-id "${sgid}"
else
  echo 'No main security group to delete'
fi
