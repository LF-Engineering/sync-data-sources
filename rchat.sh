#/bin/bash
SDS_ONLY_P2O=1 SDS_CSV_PREFIX=/root/sds-prod-rchat SDS_ES_BULKSIZE='500' SDS_DATASOURCES_RE='rocketchat' SDS_DATASOURCES_SKIP_RE='^slack$' SDS_SKIP_SSAW='1' SDS_SILENT='1' SDS_TASK_TIMEOUT_SECONDS='43200' SDS_NCPUS_SCALE='1.5' SDS_SCROLL_SIZE='500' /usr/bin/sds-cron-task.sh sds-prod-rchat prod 1>> /tmp/sds-prod-rchat.log 2>>/tmp/sds-prod-rchat.err
