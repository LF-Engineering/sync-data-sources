#!/bin/bash
if [ -z "${AWS_PROFILE}" ]
then
  echo "$0: you need to specify AWS_PROFILE=..."
  exit 1
fi
aws efs describe-file-systems | jq '.FileSystems[] | select(.Name == "sds-efs-volume")'
