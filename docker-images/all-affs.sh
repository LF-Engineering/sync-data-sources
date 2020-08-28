#/bin/bash
SDS_NO_INDEX_DROP='1' SDS_SKIP_CHECK_FREQ=1 SDS_SKIP_DATA=1 SDS_ONLY_P2O=1 SDS_CSV_PREFIX=/root/sds-all-affs SDS_TASK_TIMEOUT_SECONDS='43200' SDS_NCPUS_SCALE='1.5' SDS_SCROLL_SIZE='1000' SDS_ES_BULKSIZE='1000' /usr/bin/sds-cron-task.sh sds-all-affs prod 1>> /tmp/sds-all-affs.log 2>>/tmp/sds-all-affs.err
