#!/bin/bash
if [ -z "${AWS_PROFILE}" ]
then
  echo "$0: you need to specify AWS_PROFILE=..."
  exit 1
fi
fsid=`aws efs describe-file-systems | jq -r '.FileSystems[] | select(.Name == "sds-efs-volume") | .FileSystemId'`
aws efs delete-file-system --file-system-id "${fsid}"
