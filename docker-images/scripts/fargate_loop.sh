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
./fargate_run.sh
./fargate_run.sh
