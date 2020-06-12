#!/bin/bash
date
if [ -z "$1" ]
then
  echo "$0: you need to specify name for this run, can be 'sds1' for example"
  exit 1
fi
cd /root/go/src/github.com/LF-Engineering/sync-data-sources/ || exit 2
lock_file="/tmp/$1.lock"
function cleanup {
  rm -f "${lock_file}"
}
if [ -f "${lock_file}" ]
then
  # echo "$0: another SDS instance \"$1\" is still running, exiting"
  exit 3
fi
> "${lock_file}"
trap cleanup EXIT
./docker-images/run_sds.sh prod
