#!/bin/bash
if [ -z "${AWS_PROFILE}" ]
then
  echo "$0: you need to specify AWS_PROFILE=..."
  exit 1
fi
fsid=`aws efs describe-file-systems | jq -r '.FileSystems[] | select(.Name == "sds-efs-volume") | .FileSystemId'`
apid=`aws efs describe-access-points | jq -r '.AccessPoints[] | select(.Name == "sds-efs-access-point") | .AccessPointId'`
aws efs delete-access-point --access-point-id "${apid}"
aws efs delete-file-system --file-system-id "${fsid}"
