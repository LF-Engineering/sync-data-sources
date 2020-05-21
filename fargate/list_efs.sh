#!/bin/bash
# AP=1 - list access point too
if [ -z "${AWS_PROFILE}" ]
then
  echo "$0: you need to specify AWS_PROFILE=..."
  exit 1
fi
aws efs describe-file-systems | jq '.FileSystems[] | select(.Name == "sds-efs-volume")'
if [ ! -z "${AP}" ]
then
  aws efs describe-access-points | jq '.AccessPoints[] | select(.Name == "sds-efs-access-point")'
fi
fsid=`aws efs describe-file-systems | jq -r '.FileSystems[] | select(.Name == "sds-efs-volume") | .FileSystemId'`
aws efs describe-mount-targets --file-system-id "${fsid}" | jq '.MountTargets[]'
