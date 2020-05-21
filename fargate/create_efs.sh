#!/bin/bash
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
aws efs create-file-system --region "${AWS_REGION}" --creation-token sds-efs-volume --performance-mode maxIO --tags "Key=Name,Value=sds-efs-volume"
