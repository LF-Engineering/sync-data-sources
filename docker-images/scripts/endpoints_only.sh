#!/bin/bash
/fetch.sh
mv /data/cncf/thanos.yaml /
rm -rf /data/*
mv /thanos.yaml /data/
vim -c '%s/disabled:true//g' -c wq /data/thanos.yaml
SDS_SKIP_ES_DATA=1 SDS_SKIP_ES_LOG=1 SDS_DEBUG=1 SDS_CMDDEBUG=1 SDS_ONLY_P2O=1 SDS_SKIP_P2O='' SDS_SKIP_AFFS=1 SDS_SKIP_DATA='' DS_DRY_RUN='' syncdatasources 2>&1 | tee -a /sds.log
