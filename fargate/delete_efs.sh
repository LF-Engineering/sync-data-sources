#!/bin/bash
# AP=1 - delete access points too
if [ -z "${AWS_PROFILE}" ]
then
  echo "$0: you need to specify AWS_PROFILE=..."
  exit 1
fi
fsid=`aws efs describe-file-systems | jq -r '.FileSystems[] | select(.Name == "sds-efs-volume") | .FileSystemId'`
mtid=`aws efs describe-mount-targets --file-system-id "${fsid}" | jq -r '.MountTargets[] | .MountTargetId'`
if [ ! -z "$mtid" ]
then
  aws efs delete-mount-target --mount-target-id "${mtid}"
else
  echo 'No mount targets to delete'
fi
if [ ! -z "${AP}" ]
then
  apid=`aws efs describe-access-points | jq -r '.AccessPoints[] | select(.Name == "sds-efs-access-point") | .AccessPointId'`
  if [ ! -z "${apid}" ]
  then
    aws efs delete-access-point --access-point-id "${apid}"
  else
    echo 'No access points to delete'
  fi
fi
if [ ! -z "${fsid}" ]
then
  aws efs delete-file-system --file-system-id "${fsid}"
else
  echo 'No filesystems to delete'
fi
