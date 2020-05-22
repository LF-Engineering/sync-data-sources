#!/bin/bash
# AP=1 - delete access points too
if [ -z "${AWS_PROFILE}" ]
then
  echo "$0: you need to specify AWS_PROFILE=..."
  exit 1
fi
fsid=`aws efs describe-file-systems | jq -r '.FileSystems[] | select(.Name == "sds-efs-volume") | .FileSystemId'`
if [ ! -z "${fsid}" ]
then
  mtid=`aws efs describe-mount-targets --file-system-id "${fsid}" | jq -r '.MountTargets[] | .MountTargetId'`
fi
if [ ! -z "$mtid" ]
then
  echo "Deleting mount target ${mtid}"
  aws efs delete-mount-target --mount-target-id "${mtid}"
else
  echo 'No mount target to delete'
fi
if [ ! -z "${AP}" ]
then
  apid=`aws efs describe-access-points | jq -r '.AccessPoints[] | select(.Name == "sds-efs-access-point") | .AccessPointId'`
  if [ ! -z "${apid}" ]
  then
    echo "Deleting access point ${apid}"
    aws efs delete-access-point --access-point-id "${apid}"
  else
    echo 'No access point to delete'
  fi
fi
while true
do
  mtid=`aws efs describe-mount-targets --file-system-id "${fsid}" | jq -r '.MountTargets[] | .MountTargetId'`
  if [ -z "${mtid}" ]
  then
    break
  else
    echo 'Waiting for mount targets to disappear...'
    sleep 5
  fi
done
if [ ! -z "${fsid}" ]
then
  echo "Deleting filesystem ${fsid}"
  aws efs delete-file-system --file-system-id "${fsid}"
else
  echo 'No filesystem to delete'
fi
