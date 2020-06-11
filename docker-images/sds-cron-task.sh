#!/bin/bash
date
cd /root/go/src/github.com/LF-Engineering/sync-data-sources/ || exit 1
lock_file=/tmp/sds.lock
function cleanup {
  rm -f "${lock_file}"
}
if [ -f "${lock_file}" ]
then
  # echo "$0: another SDS instance is still running, exiting"
  exit 2
fi
> "${lock_file}"
trap cleanup EXIT
SDS_NCPUS_SCALE=2 SDS_ES_BULKSIZE=100 ./docker-images/run_sds.sh prod
