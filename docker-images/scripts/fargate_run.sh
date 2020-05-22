#!/bin/sh
if [ -z "${REPO_ACCESS}" ]
then
  echo "REPO_ACCESS env variable must be set"
  exit 1
fi
if [ -z "${SDS_TASK_NAME}" ]
then
  echo "SDS_TASK_NAME env variable must be set"
  exit 2
fi
rm -r /data /root/.perceval
ls -l /efs
ls -l "/efs/${SDS_TASK_NAME}"
mv "/efs/${SDS_TASK_NAME}" /root/.perceval
./fetch.sh && syncdatasources
mv /root/.perceval "/efs/${SDS_TASK_NAME}"
