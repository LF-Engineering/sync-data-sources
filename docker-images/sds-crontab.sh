#!/bin/bash
if [ -z "$1" ]
then
  echo "$0: you need to specify env name for this deployment: prod|test"
  exit 1
fi
cd /root/go/src/github.com/LF-Engineering/sync-data-sources/sources/ || exit 2
git pull || exit 3
./sds-crontab "$1"
