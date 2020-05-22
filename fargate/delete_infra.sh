#!/bin/bash
# AP=1 - delete with AP mode
if [ -z "${AWS_PROFILE}" ]
then
  echo "$0: you need to specify AWS_PROFILE=..."
  exit 1
fi
./fargate/delete_efs.sh
