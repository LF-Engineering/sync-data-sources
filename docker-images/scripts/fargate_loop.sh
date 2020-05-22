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
rm -r "/efs/${SDS_TASK_NAME}"
./fargate_run.sh
./fargate_run.sh
./fargate_run.sh
